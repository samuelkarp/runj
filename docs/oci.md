# OCI runtime spec notes

This document records runj's FreeBSD-specific extensions to the OCI runtime
specification, along with notes on how runj interprets parts of the spec that
are underspecified or where it follows `runc`'s behavior for compatibility.

# FreeBSD extensions

FreeBSD-specific jail configuration can be supplied in two ways, each with its
own schema:

1. Directly in the bundle's `config.json`, using the OCI runtime spec's own
   `freebsd.jail` fields.
2. In a runj-specific `runj.ext.json` file in the bundle directory, using a
   separate runj-defined schema rooted at a `network` struct.  This allows
   software that generates a `config.json` without awareness of FreeBSD or runj
   to be augmented with additional settings without modifying the generator.

When a `runj.ext.json` file is present, runj merges it into the configuration
loaded from `config.json`.

## In `config.json` (`freebsd.jail` schema)

runj reads the following fields from the OCI runtime spec's `freebsd.jail`
struct:
* `host` (string) - UTS sharing mode, covering the hostname, domainname, host
  id, and host uuid.  Valid options are `new` and `inherit`.  Equivalent to the
  `host` field described in the `jail(8)` manual page.
* `ip4` (string) - IPv4 mode.  Valid options are `new`, `inherit`, and
  `disable`.  Equivalent to the `ip4` field described in the `jail(8)` manual
  page.
* `ip4Addr` ([]string) - list of IPv4 addresses assigned to the jail.
  Equivalent to the `ip4.addr` field described in the `jail(8)` manual page.
* `ip6` (string) - IPv6 mode.  Valid options are `new`, `inherit`, and
  `disable`.  Equivalent to the `ip6` field described in the `jail(8)` manual
  page.
* `ip6Addr` ([]string) - list of IPv6 addresses assigned to the jail.
  Equivalent to the `ip6.addr` field described in the `jail(8)` manual page.
* `vnet` (string) - vnet mode.  Valid options are `new` and `inherit`.
  Equivalent to the `vnet` field described in the `jail(8)` manual page.
* `vnetInterfaces` ([]string) - list of network interfaces assigned to the jail.
  Equivalent to the `vnet.interface` field described in the `jail(8)` manual
  page.

For both IPv4 and IPv6, runj exposes only the address-family mode and the
address list.  Other `jail(8)` sub-parameters — such as `ip6.saddrsel` and
`ip4.saddrsel` — are not exposed, because the OCI `freebsd.jail` schema defines
no fields for them.

The `host` field sets the mode only.  runj takes the `host.hostname`
sub-parameter from the spec's top-level `hostname` field.  Setting
`host:inherit` and providing a value for `hostname` is invalid and is rejected
by runj.  The `host.domainname`, `host.hostid`, and `host.hostuuid`
sub-parameters are unspecified in the OCI `freebsd.jail` schema.

An example embedded in `config.json`:

```json
{
  "ociVersion": "1.3.0",
  "process": {
    // omitted
  },
  "freebsd": {
    "jail": {
      "host": "new",
      "ip4": "new",
      "ip4Addr": ["127.0.0.2"],
      "ip6": "new",
      "ip6Addr": ["::1"],
      "vnet": "new",
      "vnetInterfaces": ["epair0b"]
    }
  }
}
```

## In `runj.ext.json` (runj `network` schema)

Fields inside the `network` struct:
* `ipv4` (struct)
* `vnet` (struct)

Fields inside the `ipv4` struct:
* `mode` (string) - valid options are `new`, `inherit`, and `disable`.  This
  field is the equivalent of the `ip4` field described in the `jail(8)` manual
  page.
* `addr` ([]string) - list of IPv4 addresses assigned to the jail.  This field
  is the equivalent of the `ip4.addr` field described in the `jail(8)` manual
  page.

Fields inside the `vnet` struct:
* `mode` (string) - valid options are `new` and `inherit`.  This field is the
  equivalent of the `vnet` field described in the `jail(8)` manual page.
* `interfaces` ([]string) - list of network interfaces assigned to the jail.
  This field is the equivalent of the `vnet.interface` field described in the
  `jail(8)` manual page.

An example `runj.ext.json`:

```json
{
  "network": {
    "ipv4": {
      "mode": "inherit"
    },
    "vnet": {
      "mode": "inherit"
    }
  }
}
```
# `create`

The `create` command is documented [in the
spec](https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#create)
as requiring two positional arguments: `<container-id> <path-to-bundle>`.
However, `runc` (the reference implementation of the spec) accepts only a single
positional argument (`<container-id>`) and instead uses either the current
working directory as the bundle or accepts it through the `-b`/`--bundle` flag.
See [here](https://github.com/opencontainers/runc/blob/2cf8d240075dd322b9385100c9af4b149c973391/create.go#L12-L30).
For compatibility with runc and other integrations, runj now supports the flag
in addition to the positional argument form.

## Non-terminal STDIO

The spec does not describe how container STDIO should be handled.  runc passes
the STDIO file descriptors for the `runc create` invocation to the container
process.  Programs that invoke runc (like containerd) configure STDIO for `runc
create` and provide input/collect output for the container through those
descriptors.

## Terminal STDIO

The spec includes a `terminal` field, but does not describe how a runtime should
expose the terminal.  runc expects an `AF_UNIX` socket to be provided as a
command-line argument in the `--console-socket` field; runc's init process is
responsible for allocating a `pty` and sending the control device's file
descriptor over the provided socket.

runc opens the socket file as part of `runc create`, then passes that socket
file as an extra file descriptor when starting the init process, indicating the
fd number with the `_LIBCONTAINER_CONSOLE` environment variable.  The init
process creates the pty (`console.NewPty`),  sends the control fd number to the
socket (`utils.SendFd`), and uses `dup3(2)` to override the standard I/O file
descriptors (0, 1, and 2) for the container process.

A runc client like `containerd-shim-runc-v2` is responsible for creating the
socket passed as `--console-socket` (with `pkg/process/init.go:Create`),
receiving the control device (with `socket.ReceiveMaster`), then copying bytes
to and from the device.


# `start`

runc's implementation of the start command exits immediately after starting
the container's process.  This does not appear to be specified in the spec.
