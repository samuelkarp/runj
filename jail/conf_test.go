package jail

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderConfigGolden(t *testing.T) {
	tests := []struct {
		// name is both used as the subtest name and is the name of the golden data file in testdata
		name   string
		config Config
	}{{
		"basic",
		Config{
			Name: "basic",
			Root: "/tmp/test/basic/root",
		},
	}, {
		"hostname",
		Config{
			Name:     "hostname",
			Root:     "/tmp/test/hostname/root",
			Hostname: "test.hostname.example.com",
		},
	}, {
		"network",
		Config{
			Name:    "network",
			Root:    "/tmp/test/network/root",
			IP4:     "new",
			IP4Addr: []string{"one", "two", "three"},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := ioutil.ReadFile(filepath.Join("testdata", fmt.Sprintf("%s.conf", tc.name)))
			assert.NoError(t, err, "test data")
			actual, err := renderConfig(&tc.config)
			assert.NoError(t, err, "render")
			assert.Equal(t, string(expected), actual)
		})
	}
}
