package oci

import (
	"testing"

	"github.com/go-faker/faker/v4"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"gotest.tools/v3/assert"

	runjspec "go.sbk.wtf/runj/runtimespec"
)

func TestMergeEmpty(t *testing.T) {
	spec := &runtimespec.Spec{}
	freebsd := &runjspec.FreeBSD{}
	err := faker.FakeData(freebsd)
	assert.NilError(t, err)

	merge(spec, freebsd)
	assert.Equal(t, string(spec.FreeBSD.Jail.Vnet), string(freebsd.Network.VNet.Mode))
	assert.DeepEqual(t, spec.FreeBSD.Jail.VnetInterfaces, freebsd.Network.VNet.Interfaces)
	assert.Equal(t, string(spec.FreeBSD.Jail.Ip4), string(freebsd.Network.IPv4.Mode))
	assert.DeepEqual(t, spec.FreeBSD.Jail.Ip4Addr, freebsd.Network.IPv4.Addr)
}

// TestMergeNilArguments verifies that merge tolerates nil inputs without
// panicking and without creating spec state.
func TestMergeNilArguments(t *testing.T) {
	// nil freebsd: spec must be left untouched.
	spec := &runtimespec.Spec{}
	merge(spec, nil)
	assert.Assert(t, spec.FreeBSD == nil)

	// nil spec: must not panic.
	merge(nil, &runjspec.FreeBSD{})
}

// TestMergeNilNetwork verifies that a FreeBSD section with no network still
// establishes the FreeBSD.Jail struct but sets no networking fields.
func TestMergeNilNetwork(t *testing.T) {
	spec := &runtimespec.Spec{}
	merge(spec, &runjspec.FreeBSD{})
	assert.Assert(t, spec.FreeBSD != nil)
	assert.Assert(t, spec.FreeBSD.Jail != nil)
	assert.Equal(t, string(spec.FreeBSD.Jail.Ip4), "")
	assert.Equal(t, string(spec.FreeBSD.Jail.Vnet), "")
}

func TestMergeIPv4Only(t *testing.T) {
	spec := &runtimespec.Spec{}
	merge(spec, &runjspec.FreeBSD{
		Network: &runjspec.FreeBSDNetwork{
			IPv4: &runjspec.FreeBSDIPv4{
				Mode: runjspec.FreeBSDIPv4ModeNew,
				Addr: []string{"127.0.0.2"},
			},
		},
	})
	assert.Equal(t, string(spec.FreeBSD.Jail.Ip4), "new")
	assert.DeepEqual(t, spec.FreeBSD.Jail.Ip4Addr, []string{"127.0.0.2"})
	// VNet was not specified and must remain unset.
	assert.Equal(t, string(spec.FreeBSD.Jail.Vnet), "")
	assert.Assert(t, spec.FreeBSD.Jail.VnetInterfaces == nil)
}

func TestMergeVNetOnly(t *testing.T) {
	spec := &runtimespec.Spec{}
	merge(spec, &runjspec.FreeBSD{
		Network: &runjspec.FreeBSDNetwork{
			VNet: &runjspec.FreeBSDVNet{
				Mode:       runjspec.FreeBSDVNetModeNew,
				Interfaces: []string{"epair0b"},
			},
		},
	})
	assert.Equal(t, string(spec.FreeBSD.Jail.Vnet), "new")
	assert.DeepEqual(t, spec.FreeBSD.Jail.VnetInterfaces, []string{"epair0b"})
	assert.Equal(t, string(spec.FreeBSD.Jail.Ip4), "")
	assert.Assert(t, spec.FreeBSD.Jail.Ip4Addr == nil)
}

// TestMergeAppendsToExisting verifies that address and interface lists from the
// FreeBSD section are appended to values already present in the spec.
func TestMergeAppendsToExisting(t *testing.T) {
	spec := &runtimespec.Spec{
		FreeBSD: &runtimespec.FreeBSD{
			Jail: &runtimespec.FreeBSDJail{
				Ip4Addr:        []string{"127.0.0.1"},
				VnetInterfaces: []string{"epair0b"},
			},
		},
	}
	merge(spec, &runjspec.FreeBSD{
		Network: &runjspec.FreeBSDNetwork{
			IPv4: &runjspec.FreeBSDIPv4{Addr: []string{"10.2.2.2"}},
			VNet: &runjspec.FreeBSDVNet{Interfaces: []string{"epair1b"}},
		},
	})
	assert.DeepEqual(t, spec.FreeBSD.Jail.Ip4Addr, []string{"127.0.0.1", "10.2.2.2"})
	assert.DeepEqual(t, spec.FreeBSD.Jail.VnetInterfaces, []string{"epair0b", "epair1b"})
}

// TestMergeEmptyModePreservesExisting verifies that an empty mode in the FreeBSD
// section does not overwrite a mode already set in the spec.
func TestMergeEmptyModePreservesExisting(t *testing.T) {
	spec := &runtimespec.Spec{
		FreeBSD: &runtimespec.FreeBSD{
			Jail: &runtimespec.FreeBSDJail{Ip4: "inherit"},
		},
	}
	merge(spec, &runjspec.FreeBSD{
		Network: &runjspec.FreeBSDNetwork{
			IPv4: &runjspec.FreeBSDIPv4{Addr: []string{"10.2.2.2"}},
		},
	})
	assert.Equal(t, string(spec.FreeBSD.Jail.Ip4), "inherit")
	assert.DeepEqual(t, spec.FreeBSD.Jail.Ip4Addr, []string{"10.2.2.2"})
}
