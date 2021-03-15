*Placeholder for OCI changes*

# `create`

The `create` command is documented [in the
spec](https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#create)
as requiring two positional arguments: `<container-id> <path-to-bundle>`.
However, `runc` (the reference implementation of the spec) accepts only a single
positional argument (`<container-id>`) and instead uses either the current
working directory as the bundle or accepts it through the `-b`/`bundle` flag.
See [here](https://github.com/opencontainers/runc/blob/2cf8d240075dd322b9385100c9af4b149c973391/create.go#L12-L30).

The spec does not describe how container STDIO should be handled.  runc passes
the STDIO file descriptors for the `runc create` invocation to the container
process.  Programs that invoke runc (like containerd) configure STDIO for `runc
create` and provide input/collect output for the container through those
descriptors.

# `start`

runc's implementation of the start command exits immediately after starting
the container's process.  This does not appear to be specified in the spec.
