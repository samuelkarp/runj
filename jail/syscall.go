package jail

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	_FLAG_CREATE = 0x01
)

// ID identifies jails
type ID int32

func (id ID) String() string {
	return strconv.Itoa(int(id))
}

// attach attaches the current process to the jail with SYS_JAIL_ATTACH
func attach(jid ID) error {
	return jidSyscall(syscall.SYS_JAIL_ATTACH, jid)
}

// remove destroys the jail, killing all processes within it with SYS_JAIL_REMOVE
func remove(jid ID) error {
	return jidSyscall(syscall.SYS_JAIL_REMOVE, jid)
}

// find queries the OS for a jail with the specified name or JID
func find(identifier string) (ID, error) {
	params := &findParams{}
	jid, err := strconv.Atoi(identifier)
	if err == nil {
		if jid == 0 {
			return 0, nil
		}
		if jid > math.MaxInt32 {
			return 0, errors.New("invalid jid")
		}
		params.jid = int32(jid)
	} else {
		params.name = identifier
	}
	iovec, err := params.iovec()
	if err != nil {
		return 0, err
	}
	return get(iovec, 0)
}

type findParams struct {
	jid  int32
	name string
}

func (f *findParams) iovec() ([]syscall.Iovec, error) {
	iovec := make([]syscall.Iovec, 0)
	if f.jid != 0 {
		jidKey, err := syscall.ByteSliceFromString("jid")
		if err != nil {
			return nil, err
		}
		jidVal := (*byte)(unsafe.Pointer(&f.jid))
		iovec = append(iovec, makeIovec(jidKey, jidVal, 4)...)
	}
	if f.name != "" {
		i, err := stringIovec("name", f.name)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, i...)
	}
	return iovec, nil
}

// get calls SYS_JAIL_GET
func get(iovecs []syscall.Iovec, flags int) (ID, error) {
	return iovSyscall(syscall.SYS_JAIL_GET, iovecs, flags)
}

// set creates or modifies jails with parameters provided in []syscall.Iovec via SYS_JAIL_SET
func set(iovecs []syscall.Iovec, flags int) (ID, error) {
	return iovSyscall(syscall.SYS_JAIL_SET, iovecs, flags)
}

func jidSyscall(callnum uintptr, jid ID) error {
	_, _, errno := syscall.Syscall(callnum, uintptr(jid), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func iovSyscall(callnum uintptr, iovecs []syscall.Iovec, flags int) (ID, error) {
	errbuf, erriov := errorIovec()
	iovecs = append(iovecs, erriov...)

	jid, _, errno := syscall.Syscall(callnum, uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)), uintptr(flags))
	if int32(jid) == -1 || errno != 0 {
		if errbuf[0] == 0 {
			return ID(jid), errno
		}
		return ID(jid), fmt.Errorf("errmsg: %s", unix.ByteSliceToString(errbuf))
	}
	return ID(jid), nil
}

const (
	errorBufferLen = 1024
	errorKey       = "errmsg"
)

func errorIovec() ([]byte, []syscall.Iovec) {
	buffer := make([]byte, errorBufferLen)
	n, _ := syscall.ByteSliceFromString(errorKey)
	return buffer, makeIovec(n, &buffer[0], len(buffer))
}

func stringIovec(name string, value string) ([]syscall.Iovec, error) {
	n, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}
	v, err := syscall.BytePtrFromString(value)
	if err != nil {
		return nil, err
	}
	return makeIovec(n, v, len(value)+1), nil
}

func int32Iovec(name string, value int32) ([]syscall.Iovec, error) {
	n, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}
	v := (*byte)(unsafe.Pointer(&value))
	size := 4
	return makeIovec(n, v, size), nil
}

func netIPIovec(name string, value []netip.Addr) ([]syscall.Iovec, error) {
	n, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, 0)
	is6 := false
	for i, addr := range value {
		if i == 0 {
			is6 = addr.Is6()
		} else if is6 && addr.Is4() {
			return nil, fmt.Errorf("expected IPv6 but %v is IPv4", addr)
		} else if !is6 && addr.Is6() {
			return nil, fmt.Errorf("expected IPv4 but %v is IPv6", addr)
		}
		bytes = append(bytes, addr.AsSlice()...)
	}
	return makeIovec(n, &bytes[0], len(bytes)), nil
}

func nilIovec(name string) ([]syscall.Iovec, error) {
	n, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}
	return makeIovec(n, nil, 0), nil
}

func makeIovec(name []byte, value *byte, size int) []syscall.Iovec {
	iovecs := make([]syscall.Iovec, 2)

	iovecs[0].Base = &name[0]
	iovecs[0].SetLen(len(name))

	iovecs[1].Base = value
	iovecs[1].SetLen(size)
	return iovecs
}
