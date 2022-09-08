package oci

import (
	"testing"

	"github.com/bxcodec/faker"

	"go.sbk.wtf/runj/runtimespec"
	"gotest.tools/v3/assert"
)

func TestMergeEmpty(t *testing.T) {
	spec := &runtimespec.Spec{}
	freebsd := &runtimespec.FreeBSD{}
	err := faker.FakeData(freebsd)
	assert.NilError(t, err)

	merge(spec, freebsd)
	assert.DeepEqual(t, freebsd, spec.FreeBSD)
}
