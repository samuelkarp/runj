package jail

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderConfigBasic(t *testing.T) {
	const (
		id   = "basic"
		path = "/tmp/test/basic/root"
	)
	expected, err := ioutil.ReadFile("testdata/basic.conf")
	assert.NoError(t, err, "test data")
	actual, err := renderConfig(id, path)
	assert.NoError(t, err, "render")
	assert.Equal(t, string(expected), actual)
}
