/*
This file previously contained code from
https://github.com/opencontainers/runtime-spec/blob/e6143ca7d51d11b9ab01cf4bc39e73e744241a1b/specs-go/config.go,
retrieved October 28, 2020.

It now only contains original code.

Copyright 2020 Samuel Karp.
*/

package runtimespec

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
	FreeBSDIPv4ModeInherit FreeBSDIPv4Mode = "inherit"
	FreeBSDIPv4ModeDisable FreeBSDIPv4Mode = "disable"
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
	FreeBSDVNetModeInherit FreeBSDVNetMode = "inherit"
)

type FreeBSDVNetMode string
