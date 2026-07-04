# Spec coverage

This document tracks runj's coverage of the
[OCI runtime specification](https://github.com/opencontainers/runtime-spec),
version 1.3.0.

* `[x]` - implemented
* `[ ]` - not yet implemented

runj currently ignores configuration it does not implement rather than
returning an error.  The specification requires a runtime to error when it
cannot apply a configured property; closing that gap is itself an item below.

## Lifecycle operations

* [x] `create`
* [x] `start`
* [x] `kill`
* [x] `delete`
* [x] `state`

## Process

* [x] `process.args`
* [x] `process.env`
* [x] `process.terminal`
* [ ] `process.user` (uid, gid, umask, additionalGids) - the process runs as
  whoever invoked runj
* [ ] `process.cwd` - the working directory is hard-coded to `/`
* [ ] `process.rlimits` - tagged `linux,solaris,zos` in the spec, but
  `setrlimit(2)` applies on FreeBSD
* [ ] `process.consoleSize`

## Root

* [x] `root.path`
* [ ] `root.readonly`

## Other top-level fields

* [x] `hostname`
* [x] `mounts`
* [x] `annotations` (forwarded to hooks)
* [ ] `domainname`
* [ ] error on unsupported configuration (see note above)
* [x] honor the bundle's `ociVersion`

## Hooks

* [x] `createRuntime`
* [x] `poststop`
* [ ] `poststart`
* [ ] `createContainer`
* [ ] `startContainer`
* [ ] `prestart` (deprecated in the spec)

## FreeBSD (`freebsd.*`)

* [x] `jail.ip4`
* [x] `jail.ip4Addr`
* [x] `jail.vnet`
* [x] `jail.vnetInterfaces` (moved with `ifconfig(8)`, not set as a jail param)
* [ ] `jail.ip6`, `jail.ip6Addr` - no IPv6 support
* [ ] `jail.allow.*` - capability toggles (`setHostname`, `rawSockets`,
  `chflags`, `mount`, `quotas`, `socketAf`, `mlock`, `reservedPorts`, `suser`)
* [ ] `freebsd.devices` - individual device nodes (only whole-`devfs` mounts
  work today)
* [ ] `jail.parent` - parent jail / shared vnet
* [ ] `jail.host` - UTS sharing mode
* [ ] `jail.interface` - interface for `ip4Addr`/`ip6Addr`
* [ ] `jail.sysvmsg`, `jail.sysvsem`, `jail.sysvshm` - SystemV IPC sharing
* [ ] `jail.enforceStatfs` - mount visibility

## Resource limits

* [ ] kernel `rctl(8)` limits (the FreeBSD analogue to Linux
  `linux.resources`)
