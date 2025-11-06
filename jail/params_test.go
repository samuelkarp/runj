package jail

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

type fakeIovec struct {
	name string
	val  []byte
}

func TestCreateParamsIovec(t *testing.T) {
	tests := []struct {
		name   string
		config CreateParams
		iovec  []fakeIovec
		err    error
	}{{
		name: "basic",
		config: CreateParams{
			Name: "basic",
			Root: "/tmp/test/basic/root",
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("basic\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/basic/root\x00"),
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "hostname",
		config: CreateParams{
			Name:     "hostname",
			Root:     "/tmp/test/hostname/root",
			Hostname: "test.hostname.example.com",
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("hostname\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/hostname/root\x00"),
		}, {
			name: "host.hostname\x00",
			val:  []byte("test.hostname.example.com\x00"),
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "ip4-network",
		config: CreateParams{
			Name:    "network",
			Root:    "/tmp/test/network/root",
			IP4:     "new",
			IP4Addr: []string{"127.0.0.1", "10.2.2.2", "3.3.3.3"},
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("network\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/network/root\x00"),
		}, {
			name: "ip4\x00",
			val:  []byte{1, 0, 0, 0},
		}, {
			name: "ip4.addr\x00",
			val:  []byte{127, 0, 0, 1, 10, 2, 2, 2, 3, 3, 3, 3},
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "ip4-inherit",
		config: CreateParams{
			Name: "network",
			Root: "/tmp/test/network/root",
			IP4:  "inherit",
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("network\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/network/root\x00"),
		}, {
			name: "ip4\x00",
			val:  []byte{2, 0, 0, 0},
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "ip4-disable",
		config: CreateParams{
			Name: "network",
			Root: "/tmp/test/network/root",
			IP4:  "disable",
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("network\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/network/root\x00"),
		}, {
			name: "ip4\x00",
			val:  []byte{0, 0, 0, 0},
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "vnet",
		config: CreateParams{
			Name:          "vnet",
			Root:          "/tmp/test/vnet/root",
			VNet:          "new",
			VNetInterface: []string{"epair0b", "epair1b"},
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("vnet\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/vnet/root\x00"),
		}, {
			name: "vnet\x00",
			val:  []byte{1, 0, 0, 0},
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "vnet-inherit",
		config: CreateParams{
			Name: "vnet",
			Root: "/tmp/test/vnet/root",
			VNet: "inherit",
		},
		iovec: []fakeIovec{{
			name: "name\x00",
			val:  []byte("vnet\x00"),
		}, {
			name: "path\x00",
			val:  []byte("/tmp/test/vnet/root\x00"),
		}, {
			name: "vnet\x00",
			val:  []byte{2, 0, 0, 0},
		}, {
			name: "persist\x00",
		}},
	}, {
		name: "ip4.addr-invalid",
		config: CreateParams{
			Name:    "ip4.addr-invalid",
			IP4Addr: []string{"one"},
		},
		err: errors.New(`jail: failed to parse "one" as IPv4: ParseAddr("one"): unable to parse IP`),
	}, {
		name: "ip4-invalid",
		config: CreateParams{
			Name: "ip4-invalid",
			IP4:  "foobar",
		},
		err: errors.New(`jail: unknown IP4 type "foobar"`),
	}, {
		name: "vnet-invalid",
		config: CreateParams{
			Name: "vnet-invalid",
			VNet: "disable",
		},
		err: errors.New(`jail: unknown VNet type "disable"`),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.config.iovec()
			if tc.err != nil {
				assert.Error(t, err, tc.err)
				assert.Equal(t, tc.err.Error(), err.Error())
				return
			}
			assert.NoError(t, err, "iovec")
			converted, err := toFakeIovec(actual)
			assert.NoError(t, err, "toFakeIovec")
			assert.EqualValues(t, tc.iovec, converted)
		})
	}
}

func toFakeIovec(actual []syscall.Iovec) ([]fakeIovec, error) {
	if len(actual)%2 != 0 {
		return nil, fmt.Errorf("expected even number of iovecs, got %d", len(actual))
	}
	iovecs := make([]fakeIovec, 0)
	for i := 0; i < len(actual); i += 2 {
		f, err := toSingleFakeIovec(actual[i : i+2])
		if err != nil {
			return nil, err
		}
		iovecs = append(iovecs, *f)
	}
	return iovecs, nil
}

func toSingleFakeIovec(actual []syscall.Iovec) (*fakeIovec, error) {
	if len(actual) != 2 {
		return nil, fmt.Errorf("cannot convert len([]syscall.Iovec) = %d to fakeIovec", len(actual))
	}
	first := actual[0]
	n := make([]byte, first.Len)
	for i := uint64(0); i < first.Len; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(first.Base)) + uintptr(i)))
		n[i] = b
	}
	second := actual[1]
	var v []byte
	if second.Len > 0 {
		v = make([]byte, second.Len)
		for i := uint64(0); i < second.Len; i++ {
			b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(second.Base)) + uintptr(i)))
			v[i] = b
		}
	}
	return &fakeIovec{
		name: string(n),
		val:  v,
	}, nil
}
