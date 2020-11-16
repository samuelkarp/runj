# Security

runj is a proof-of-concept and the implementation has not been evaluated for
its security.  Do not use runj on a production system.  Do not run workloads
inside runj that rely on a secure configuration.

With that said, this document attempts to describe the security-related choices
that have been made in runj.

## Directories
runj makes use of a state directory located at `/var/lib/runj`.  Directories for
individual jails exist underneath this one and contain a `jail.conf(5)` file as
well as a copy of the OCI configuration provided in the bundle.  

## Default jail configuration

### Names
Jails are identified by a name and an ID (JID).  runj uses the user-supplied
ID parameter as the jail's name and receives an automatically-assigned JID.

### Persistence
Jails are started with the "persist" directive to `jail(8)` in the
`jail.conf(5)` file.  This allows jails to exist without any running processes.

### Mounts
By default, runj adds a devfs mount with the `devfsrules_jail=4` ruleset.  This
is added to allow basic devices like `null`, `random`, and STDIO to be available
inside the jail.  (Some tools like `ps` have a dependency on `/dev/null` to
function.)

## Dependencies

### On the system
runj uses FreeBSD's userland utilities for managing jails; it does not directly
invoke the jail-related syscalls.  You must have working versions of `jail(8)`,
`jls(8)`, `jexec(8)`, and `ps(1)` installed on your system.

The default behaviors of these utilities are used in `runj`.

### Inside the jail
`runj kill` makes use of the `kill(1)` command inside the jail's rootfs; if this
command does not exist (or is not functional), `runj kill` will not work.  If
the `kill` command has been replaced by a malicious binary, invoking `runj kill`
will cause that binary to run instead of the normal `kill` command.


