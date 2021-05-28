/*
This file is adapted from
https://github.com/opencontainers/runtime-spec/blob/e6143ca7d51d11b9ab01cf4bc39e73e744241a1b/specs-go/version.go,
retrieved October 28, 2020.

Copyright 2015 The Linux Foundation.
Copyright 2020 Samuel Karp.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtimespec

import "fmt"

const (
	// VersionMajor is for an API incompatible changes
	VersionMajor = 1
	// VersionMinor is for functionality in a backwards-compatible manner
	VersionMinor = 0
	// VersionPatch is for backwards-compatible bug fixes
	VersionPatch = 2

	// Modification by Samuel Karp
	// VersionDev indicates development branch. Releases will be empty string.
	VersionDev = "-runj-dev"
	// End of modification
)

// Version is the specification version that the package types support.
var Version = fmt.Sprintf("%d.%d.%d%s", VersionMajor, VersionMinor, VersionPatch, VersionDev)
