package jail

import (
	"errors"
	"math"
	"strconv"
	"syscall"
	"unsafe"
)

// ID identifies jails
type ID int32

// attach attaches the current process to the jail
func attach(jid ID) error {
	return jidSyscall(syscall.SYS_JAIL_ATTACH, jid)
}

// remove destroys the jail, killing all processes within it
func remove(jid ID) error {
	return jidSyscall(syscall.SYS_JAIL_REMOVE, jid)
}

// find queries the OS for a jail with the specified name or JID
func find(identifier string) (ID, error) {
	params := &findIovec{}
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
	iovec, err := params.serialize()
	if err != nil {
		return 0, err
	}
	return get(iovec, 0)
}

type findIovec struct {
	jid  int32
	name string
}

func (f *findIovec) serialize() ([]syscall.Iovec, error) {
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
		nameKey, err := syscall.ByteSliceFromString("name")
		if err != nil {
			return nil, err
		}
		nameVal, err := syscall.BytePtrFromString(f.name)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, makeIovec(nameKey, nameVal, len(f.name)+1)...)
	}
	return iovec, nil
}

func makeIovec(name []byte, value *byte, valuesize int) []syscall.Iovec {
	iovecs := make([]syscall.Iovec, 2)

	iovecs[0].Base = &name[0]
	iovecs[0].SetLen(len(name))

	iovecs[1].Base = value
	iovecs[1].SetLen(valuesize)
	return iovecs
}

// get calls SYS_JAIL_GET
func get(iovecs []syscall.Iovec, flags int) (ID, error) {
	return iovSyscall(syscall.SYS_JAIL_GET, iovecs, flags)
}

func jidSyscall(callnum uintptr, jid ID) error {
	_, _, errno := syscall.Syscall(callnum, uintptr(jid), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func iovSyscall(callnum uintptr, iovecs []syscall.Iovec, flags int) (ID, error) {
	jid, _, errno := syscall.Syscall(callnum, uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)), uintptr(flags))
	if int32(jid) == -1 || errno != 0 {
		return ID(jid), errno
	}
	return ID(jid), nil
}
