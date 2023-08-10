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

	// Hostname configures the container's hostname.
	Hostname string `json:"hostname,omitempty"`

	// Mounts configures additional mounts (on top of Root).
	Mounts []Mount `json:"mounts,omitempty"`

	// Hooks configures callbacks for container lifecycle events.
	Hooks *Hooks `json:"hooks,omitempty"`

	// Annotations contains arbitrary metadata for the container.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Modification by Samuel Karp
	FreeBSD *FreeBSD `json:"freebsd,omitempty"`
	// End of modification

	// Modification by Samuel Karp
	/*
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

// Hook specifies a command that is run at a particular event in the lifecycle of a container
type Hook struct {
	Path    string   `json:"path"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	Timeout *int     `json:"timeout,omitempty"`
}

// Hooks specifies a command that is run in the container at a particular event in the lifecycle of a container
// Hooks for container setup and teardown
type Hooks struct {
	// Modification by Artem Khramov
	/*
	   // Prestart is Deprecated. Prestart is a list of hooks to be run before the container process is executed.
	   // It is called in the Runtime Namespace
	   Prestart []Hook `json:"prestart,omitempty"`
	*/
	// End of modification
	// CreateRuntime is a list of hooks to be run after the container has been created but before pivot_root or any equivalent operation has been called
	// It is called in the Runtime Namespace
	CreateRuntime []Hook `json:"createRuntime,omitempty"`
	// Modification by Artem Khramov
	/*
	   // CreateContainer is a list of hooks to be run after the container has been created but before pivot_root or any equivalent operation has been called
	   // It is called in the Container Namespace
	   CreateContainer []Hook `json:"createContainer,omitempty"`
	   // StartContainer is a list of hooks to be run after the start operation is called but before the container process is started
	   // It is called in the Container Namespace
	   StartContainer []Hook `json:"startContainer,omitempty"`
	   // Poststart is a list of hooks to be run after the container process is started.
	   // It is called in the Runtime Namespace
	   Poststart []Hook `json:"poststart,omitempty"`
	*/
	// End of modification
	// Poststop is a list of hooks to be run after the container process exits.
	// It is called in the Runtime Namespace
	Poststop []Hook `json:"poststop,omitempty"`
}

// Modification by Samuel Karp

// FreeBSD specifies FreeBSD-specific configuration options
type FreeBSD struct {
	Network *FreeBSDNetwork `json:"network,omitempty"`
}

// FreeBSDNetwork specifies how the jail's network should be configured by the
// kernel
type FreeBSDNetwork struct {
	IPv4 *FreeBSDIPv4 `json:"ipv4,omitempty"`
	VNet *FreeBSDVNet `json:"vnet,omitempty"`
}

// FreeBSDIPv4 encapsulates IPv4-specific jail options
type FreeBSDIPv4 struct {
	// Mode specifies the IPv4 mode of the jail.  Possible values are "new",
	// "inherit", and "disable".  Setting the Addr parameter implies a value of
	// "new".
	Mode FreeBSDIPv4Mode `json:"mode,omitempty"`
	// Addr is a list of IPv4 addresses assigned to the jail.  If this is set,
	// the jail is restricted to using only these addresses.
	Addr []string `json:"addr,omitempty"`
}

// FreeBSDIPv4Mode describes the mode of IPv4 in the jail.  Possible values are
// "new", "inherit", and "disable".  Setting the IPv4 Addr parameter implies a
// value of "new".
type FreeBSDIPv4Mode string

const (
	FreeBSDIPv4ModeNew     FreeBSDIPv4Mode = "new"
	FreeBSDIPv4ModeInherit                 = "inherit"
	FreeBSDIPv4ModeDisable                 = "disable"
)

// FreeBSDVNet encapsulates vnet-specific jail options
type FreeBSDVNet struct {
	// Mode specifies the vnet mode of the jail.  Possible values are "new" and
	// "inherit".  Setting the Interfaces parameter implies a value of "new".
	Mode FreeBSDVNetMode `json:"mode,omitempty"`
	// Interfaces is a list of interfaces assigned to the jail.  If this is set,
	// the interfaces are moved into the jail and are inaccessible from the
	// host.
	Interfaces []string `json:"interfaces,omitempty"`
}

const (
	FreeBSDVNetModeNew     FreeBSDVNetMode = "new"
	FreeBSDVNetModeInherit                 = "inherit"
)

type FreeBSDVNetMode string

// End of modification

// Modification by Samuel Karp
/*
Omitted type definitions for:
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
