/*
This file is adapted from
https://github.com/opencontainers/runtime-spec/blob/e6143ca7d51d11b9ab01cf4bc39e73e744241a1b/specs-go/config.go,
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

// Spec is the base configuration for the container.
type Spec struct {
	// Version of the Open Container Initiative Runtime Specification with which the bundle complies.
	Version string `json:"ociVersion"`
	// Process configures the container process.
	Process *Process `json:"process,omitempty"`
	// Root configures the container's root filesystem.
	Root *Root `json:"root,omitempty"`

	// Modification by Samuel Karp
	/*
		// Hostname configures the container's hostname.
		Hostname string `json:"hostname,omitempty"`
	*/
	// Mounts configures additional mounts (on top of Root).
	Mounts []Mount `json:"mounts,omitempty"`
	/*
		// Hooks configures callbacks for container lifecycle events.
		Hooks *Hooks `json:"hooks,omitempty" platform:"linux,solaris"`
		// Annotations contains arbitrary metadata for the container.
		Annotations map[string]string `json:"annotations,omitempty"`

		// Linux is platform-specific configuration for Linux based containers.
		Linux *Linux `json:"linux,omitempty" platform:"linux"`
		// Solaris is platform-specific configuration for Solaris based containers.
		Solaris *Solaris `json:"solaris,omitempty" platform:"solaris"`
		// Windows is platform-specific configuration for Windows based containers.
		Windows *Windows `json:"windows,omitempty" platform:"windows"`
		// VM specifies configuration for virtual-machine-based containers.
		VM *VM `json:"vm,omitempty" platform:"vm"`
	*/
	// End of modification
}

// Modification by Samuel Karp
/*
Omitted type definitions for:
LinuxCapabilities
Box
User
*/
// End of modification

// Process contains information to start a specific application inside the container.
type Process struct {
	// Terminal creates an interactive terminal for the container.
	Terminal bool `json:"terminal,omitempty"`
	// Modification by Samuel Karp
	/*
		// ConsoleSize specifies the size of the console.
		ConsoleSize *Box `json:"consoleSize,omitempty"`
		// User specifies user information for the process.
		User User `json:"user"`
	*/
	// End of modification

	// Args specifies the binary and arguments for the application to execute.
	Args []string `json:"args,omitempty"`

	// Modification by Samuel Karp
	/*
		// CommandLine specifies the full command line for the application to execute on Windows.
		CommandLine string `json:"commandLine,omitempty" platform:"windows"`
	*/

	// Env populates the process environment for the process.
	Env []string `json:"env,omitempty"`

	// Modification by Samuel Karp`
	/*
		// Cwd is the current working directory for the process and must be
		// relative to the container's root.
		Cwd string `json:"cwd"`
		// Capabilities are Linux capabilities that are kept for the process.
		Capabilities *LinuxCapabilities `json:"capabilities,omitempty" platform:"linux"`
		// Rlimits specifies rlimit options to apply to the process.
		Rlimits []POSIXRlimit `json:"rlimits,omitempty" platform:"linux,solaris"`
		// NoNewPrivileges controls whether additional privileges could be gained by processes in the container.
		NoNewPrivileges bool `json:"noNewPrivileges,omitempty" platform:"linux"`
		// ApparmorProfile specifies the apparmor profile for the container.
		ApparmorProfile string `json:"apparmorProfile,omitempty" platform:"linux"`
		// Specify an oom_score_adj for the container.
		OOMScoreAdj *int `json:"oomScoreAdj,omitempty" platform:"linux"`
		// SelinuxLabel specifies the selinux context that the container process is run as.
		SelinuxLabel string `json:"selinuxLabel,omitempty" platform:"linux"`
	*/
	// End of modification
}

// Root contains information about the container's root filesystem on the host.
type Root struct {
	// Path is the absolute path to the container's root filesystem.
	Path string `json:"path"`

	// Modification by Samuel Karp
	/*
		// Readonly makes the root filesystem for the container readonly before the process is executed.
		Readonly bool `json:"readonly,omitempty"`
	*/
	// End of modification
}

// Mount specifies a mount for a container.
type Mount struct {
	// Destination is the absolute path where the mount will be placed in the container.
	Destination string `json:"destination"`
	// Type specifies the mount kind.
	Type string `json:"type,omitempty" platform:"linux,solaris"`
	// Source specifies the source path of the mount.
	Source string `json:"source,omitempty"`
	// Options are fstab style mount options.
	Options []string `json:"options,omitempty"`
}

// Modification by Samuel Karp
/*
Omitted type definitions for:
Hook
Hooks
Linux
LinuxNamespace
LinuxNamespaceType
LinuxIDMapping
POSIXRlimit
LinuxHugepageLimit
LinuxInterfacePriority
linuxBlockIODevice
LinuxWeightDevice
LinuxThrottleDevice
LinuxBlockIO
LinuxMemory
LinuxCPU
LinuxPids
LinuxNetwork
LinuxRdma
LinuxResources
LinuxDevice
LinuxDeviceCgroup
LinuxPersonalityDomain
LinuxPersonalityFlag
LinuxPersonality
Solaris
SolarisCappedCPU
SolarisCappedMemory
SolarisAnet
Windows
WindowsDevice
WindowsResources
WindowsMemoryResources
WindowsCPUResources
WindowsStorageResources
WindowsNetwork
WindowsHyperV
VM
VMHypervisor
VMKernel
VMImage
LinuxSeccomp
Arch
LinuxSeccompFlag
LinuxSeccompAction
LinuxSeccompOperator
LinuxSeccompArg
LinuxSyscall
LinuxIntelRdt
*/
// End of modification
