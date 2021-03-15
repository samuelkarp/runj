# `jail(8)`, `jls(8)`, and `jexec(8)`

`jail(8)`, `jls(8)`, and `jexec(8)` are FreeBSD tools for jail administration.
These tools are part of a standard FreeBSD installation and are convenient ways
to interact with the relevant FreeBSD jail-related syscalls without implementing
the syscalls yourself.  `runj` uses these tools.  This document serves as a set
of notes for how the tools are used.

## `jail(8)`
* Create the `jail.conf(5)` file with `persist = true`.  This allows the jail
  object in the kernel to be created without running processes.  Create the jail
  synchronously as part of the OCI `create` command.

## `jexec(8)`

* Set environment variables prior to invoking `jail(8)`, which passes its own
  environment through when creating the jail.
* The standard I/O streams (stdio) used for `jexec(8)` are ultimately passed
  through to the jailed process.
