# containerd

This repository contains a containerd shim that can be used to run runj jails.
The containerd-shim-runj-v1 binary must be installed into `PATH`, and the
runtime should be specified as `wtf.sbk.runj.v1` when creating containers with
containerd.  Using the `ctr` tool, the runtime can be set with the `--runtime`
flag.

## Implementation details

### Number of shims
The containerd v2 shim interface allows for a shim to make its own determination
for how many shim processes should be in-use.  For the `wtf.sbk.runj.v1` shim
here, the initial design uses one shim process per container to simplify the
logic.  This may be adjusted later.
