runj
Copyright 2020-2023 Samuel Karp

This project bundles portions of the Open Container Initiative runtime
specification under the Apache License, 2.0.  For details, see the "runtimespec"
directory.

This project includes portions of runc, the reference implementation of the OCI
runtime specification, under the Apache License, 2.0.  runc includes software
originally developed at Docker, Inc. (http://www.docker.com).

When compiled as a binary and statically linked, this project bundles libraries
with separate terms.  You can use https://github.com/google/go-licenses to
determine the libraries included.  This file is also kept up to date.
{{ range . }}
=^..^=   =^..^=   =^..^=   =^..^=   =^..^=   =^..^=   =^..^=   =^..^=   =^..^=
## {{ .Name }}

* Name: {{ .Name }}
* Version: {{ .Version }}
* License: [{{ .LicenseName }}]({{ .LicenseURL }})

```
{{ .LicenseText }}
```
{{ end }}
