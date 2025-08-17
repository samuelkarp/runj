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
