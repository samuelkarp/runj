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

## Exec
The OCI spec does not define an "exec" command to execute a new process inside a
container.  However, containerd and other container runtimes expect to use such
functionality.  runc implements "exec" and containerd uses it.

* Shim Exec() creates an in-memory exec process struct associated with a given
  string ID
* Shim Start() takes the ID and invokes "runc exec" after setting up IO/console
* "runc exec" sets up IO and starts a process inside the container

## containerd bugs?

### Race in `TaskManager.Create`
`TaskManager.Create` is responsible for coordinating the logic around running
tasks in containerd.  This includes setting up the OCI bundle, launching the
shim, and calling the shim's functions.  This method contains error-handling
logic, but it looks like there's a race condition around bundle deletion and
calling the shim's `delete` command.

A somewhat simplified version of the affected logic is below:
```go
bundle, err := NewBundle(ctx, m.root, m.state, id, opts.Spec.Value)
if err != nil {
	return nil, err
}
defer func() {
	if err != nil {
		bundle.Delete()
	}
}()
// [omitted]
b := shimBinary(ctx, bundle, opts.Runtime, m.containerdAddress, m.containerdTTRPCAddress, m.events, m.tasks)
shim, _ := b.Start(ctx, topts, func() {
	log.G(ctx).WithField("id", id).Info("shim disconnected")

	cleanupAfterDeadShim(context.Background(), id, ns, m.tasks, m.events, b)
	// [omitted]
})
// [omitted]
defer func() {
	if err != nil {
		// [omitted]
		_, errShim := shim.Delete(dctx)
		if errShim != nil {
			shim.Shutdown(ctx)
			shim.Close()
		}
	}
}()
t, err := shim.Create(ctx, opts)
```

The key things to notice here are:
1. The `defer`red `bundle.Delete()` call
2. The `defer`red `shim.Delete()` and `shim.Shutdown()`
3. The call to `cleanupAfterDeadShim()` which is passed as a function pointer

The race appears to be this:
1. The call to `shim.Create` at the bottom of the snippet above fails (for
   whatever reason, though "developing a new shim" is a pretty reasonable
   reason).
2. The deferred call to `shim.Delete()` occurs, which might fail (again,
   "developing a new shim" might be why"), causing a call to `shim.Shutdown`
   where the shim exits and the ttrpc connection is severed.
3. (race) The deferred call to `bundle.Delete()` occurs, removing the bundle
   directory.
4. (race) The function pointer with the call to `cleanupAfterDeadShim` executes
   as the ttrpc connection is broken, invoking the shim's `delete` command.
   Except on Windows, the shim is always invoked with its working directory set
   to the bundle directory.  The implementation of the `delete` command is
   `shim.Shim.Cleanup`, and the built-in v2 shims on Linux both call
   `os.Getwd()` to know where the bundle is located.
5. :boom: On FreeBSD, `getwd(2)` fails with `ENOENT` when a component of the
   pathname no longer exists.  A shim imitating the behavior of the built-in
   Linux shims then may exit with an error, and containerd may output a warning
   into the log with this text: "failed to clean up after shim disconnected".
6. :boom:

A workaround appears to be that the shim should not call `os.Getwd` but instead
read the `-bundle` command-line argument.  However, the race between
`bundle.Delete()` and the shim's `delete` command will still exist.
