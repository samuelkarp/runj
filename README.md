# runj

runj is an experimental, proof-of-concept
[OCI](https://opencontainers.org)-compatible runtime for FreeBSD jails.

> **Important**: runj is a proof-of-concept and the implementation has not been
> evaluated for its security.  Do not use runj on a production system.  Do not
> run workloads inside runj that rely on a secure configuration.  This is a
> personal project, not backed by the author's employer.

## Status

[![Build Status](https://api.cirrus-ci.com/github/samuelkarp/runj.svg?branch=main)](https://cirrus-ci.com/github/samuelkarp/runj)

runj is in early development and is functional, but has very limited features.

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
  - Process environment
  - Process terminal
  - Mounts

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

### containerd

Along with the main `runj` OCI runtime, this repository also contains an
experimental shim that can be used with containerd.  The shim is available as
`containerd-shim-runj-v1` and can be used from the `ctr` command-line tool by
specifying `--runtime wtf.sbk.runj.v1`.

containerd 1.5 or later is required as earlier versions do not have all the
necessary patches for FreeBSD support.  Additional functionality may be
available in a development build of containerd.  If you prefer to build from
source, you can find the latest commits in the
[`main` branch of containerd](https://github.com/containerd/containerd/tree/main).

#### OCI Image

A base OCI image for FreeBSD 12.1-RELEASE on the `amd64` architecture is
available in the
[Amazon ECR public gallery](https://gallery.ecr.aws/samuelkarp/freebsd).  You
can pull the image with the `ctr` tool like this:

```
$ sudo ctr image pull public.ecr.aws/samuelkarp/freebsd:12.1-RELEASE
```

If you prefer to build your own image, need an image for a different
architecture, or want to try out a different version of FreeBSD, `runj` contains
a utility that can convert a FreeBSD root filesystem into an OCI image.  You
can download, convert, and import an image as follows:

```
$ runj demo download --output rootfs.txz
Found arch:  amd64
Found version:  12.1-RELEASE
Downloading image for amd64 12.1-RELEASE into rootfs.txz
[...output elided...]
$ runj demo oci-image --input rootfs.txz
Creating OCI image in file image.tar
extracting...
compressing...
computing layer digest...
writing blob sha256:f585dd296aa9697b5acaf9db7b40701a6377a3ccf4d29065cbfd3d2b80395733
writing blob sha256:413cc9413157f822242a4bb2c86ea50d20b8343964b5cf1d86182e132b51f78b
tar...
$ sudo ctr image import --index-name freebsd image.tar
unpacking freebsd (sha256:5ac2e259d1e84a9b955f7630ef499c8b6896f8409b6ac9d9a21542cb883387c0)...done
```

#### Running containers with `ctr`

With containerd, `runj`, and the `containerd-shim-runj-v1` binary installed, you
can use the `ctr` command-line tool to run containers like this:

```
$ sudo ctr run \
    --runtime wtf.sbk.runj.v1 \
    --rm \
    public.ecr.aws/samuelkarp/freebsd:12.1-RELEASE \
    my-container \
    sh -c 'echo "Hello from the container!"'
Hello from the container!
```

## Implementation details

runj uses FreeBSD's userland utilities for managing jails; it does not directly
invoke the jail-related syscalls.  You must have working versions of `jail(8)`,
`jls(8)`, `jexec(8)`, and `ps(1)` installed on your system.  `runj kill` makes
use of the `kill(1)` command inside the jail's rootfs; if this command does not
exist (or is not functional), `runj kill` will not work.

## Contributing

Please see the [contribution policy](CONTRIBUTING.md).

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
