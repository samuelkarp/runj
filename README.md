# runj

runj is an experimental, proof-of-concept
[OCI](https://opencontainers.org)-compatible runtime for FreeBSD jails.

> **Important**: runj is a proof-of-concept and the implementation has not been
> evaluated for its security.  Do not use runj on a production system.  Do not
> run workloads inside runj that rely on a secure configuration.  This is a
> personal project, not backed by the author's employer.

## Status

runj is in early development and exists primarily as a skeleton.

runj currently supports the following parts of the OCI runtime spec:

* Commands
  - Create
  - Delete
  - Start
  - State
  - Kill
* Config
  - Root path
  - Process args

## Getting started

### OCI bundle

To run a jail with runj, you must prepare an OCI bundle.  Bundles consist of a
root filesystem and a JSON-formatted configuration file.

#### Root filesystem

The root filesystem can consist either of a regular FreeBSD userland or a
reduced set of FreeBSD-compatible programs.  For experimentation, 
statically-linked programs from `/recovery` may be copied into your bundle.  You
can obtain a regular FreeBSD userland suitable for use with runj from
`http://ftp.freebsd.org/pub/FreeBSD/releases/$ARCH/$VERSION/base.txz` (where
`$ARCH` and `$VERSION` are replaced by your architecture and desired version
respectively).  Several `demo` convenience commands have been provided in runj
to assist in experimentation; you can use `runj demo download` to retrieve a
working root filesystem from the FreeBSD website.

#### Config

`runj` supports a limited number of configuration parameters for jails.
The OCI runtime spec does not currently include support for FreeBSD.  As this
proof-of-concept is developed, FreeBSD-related configuration parameters can be
added to the upstream specification.  For now, the extensions are documented
[here](docs/oci.md)

You can use `runj demo spec` to generate an example config file for your bundle.

Once you have a config file, edit the root path and process args to your desired
values.

#### Lifecycle

Create a container with `runj create $ID $BUNDLE` where `$ID` is the identifier
you picked for your container and `$BUNDLE` is the bundle directory with a valid
`config.json`.

Start your container with `runj start $ID`.  The process defined in the
`config.json` will be started.

Inspect the state of your container with `runj state $ID`.

Send a signal to your container process (or all processes in the container) with
`runj kill $ID`.

Remove your container with `runj delete $ID`.

## Implementation details

runj uses FreeBSD's userland utilities for managing jails; it does not directly
invoke the jail-related syscalls.  You must have working versions of `jail(8)`,
`jls(8)`, `jexec(8)`, and `ps(1)` installed on your system.  `runj kill` makes
use of the `kill(1)` command inside the jail's rootfs; if this command does not
exist (or is not functional), `runj kill` will not work.

## Future

Resource limits on FreeBSD can be configured using the kernel's RCTL interface.
runj does not currently use this, but may add support for it via `rctl(8)` in
the future.

## License

runj itself is licensed under the same license as the FreeBSD project.  Some
dependencies are licensed under other terms.  The OCI runtime specification and
reference code is licensed under the Apache License, 2.0; copies of that
reference code incorporated and modified in this repository remain under the
original license.
