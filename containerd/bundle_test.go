package containerd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterIncompatibleLinuxMounts(t *testing.T) {
	tests := []struct {
		in  []specs.Mount
		out []specs.Mount
	}{{
		in:  make([]specs.Mount, 0),
		out: nil,
	}, {
		in:  []specs.Mount{{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}}},
		out: []specs.Mount{{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}}},
	}, {
		in: []specs.Mount{{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
			Options:     []string{"nosuid", "noexec", "nodev"},
		}},
		out: nil,
	}, {
		in: []specs.Mount{{
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
		}, {Destination: "foo"}},
		out: []specs.Mount{{Destination: "foo"}},
	}, {
		in: []specs.Mount{{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
			Options:     []string{"nosuid", "noexec", "nodev"},
		}, {
			Destination: "/dev",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
		}, {
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
		}, {
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Source:      "shm",
			Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
		}, {
			Destination: "/dev/mqueue",
			Type:        "mqueue",
			Source:      "mqueue",
			Options:     []string{"nosuid", "noexec", "nodev"},
		}, {
			Destination: "/sys",
			Type:        "sysfs",
			Source:      "sysfs",
			Options:     []string{"nosuid", "noexec", "nodev", "ro"},
		}, {
			Destination: "/run",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
		}},
		out: nil,
	}}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			dir, err := ioutil.TempDir("", "TestFilterIncompatibleLinuxMounts")
			require.NoError(t, err, "failed to setup test dir")
			defer func() {
				err := os.RemoveAll(dir)
				assert.NoError(t, err, "failed to clean up test dir")
			}()
			inBytes, err := json.Marshal(&specs.Spec{Mounts: tc.in})
			require.NoError(t, err, "failed to marshal input")
			configJSON := filepath.Join(dir, "config.json")
			err = ioutil.WriteFile(configJSON, inBytes, 0644)
			require.NoError(t, err, "failed to write config.json")

			err = filterIncompatibleLinuxMounts(dir)
			require.NoError(t, err, "failed filter")

			out := &specs.Spec{}
			outBytes, err := ioutil.ReadFile(configJSON)
			require.NoError(t, err, "failed to read config.json")
			err = json.Unmarshal(outBytes, out)
			require.NoError(t, err, "failed to unmarshal config.json")

			assert.EqualValues(t, tc.out, out.Mounts)
		})
	}
}

func TestEqualMounts(t *testing.T) {
	tests := []struct {
		a     specs.Mount
		b     specs.Mount
		equal bool
	}{{
		a:     specs.Mount{},
		b:     specs.Mount{},
		equal: true,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		equal: true,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{},
		equal: false,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=3"}},
		equal: false,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs"},
		equal: false,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "/dev", Options: []string{"ruleset=3"}},
		equal: false,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev", Type: "fdescfs", Source: "devfs", Options: []string{"ruleset=3"}},
		equal: false,
	}, {
		a:     specs.Mount{Destination: "/dev", Type: "devfs", Source: "devfs", Options: []string{"ruleset=4"}},
		b:     specs.Mount{Destination: "/dev/fd", Type: "devfs", Source: "devfs", Options: []string{"ruleset=3"}},
		equal: false,
	}}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			equal := equalMounts(tc.a, tc.b)
			assert.Equal(t, tc.equal, equal)
		})
	}
}
