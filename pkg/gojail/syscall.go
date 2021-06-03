package gojail

import (
	"errors"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

//JailID is used to identify jails
type JailID int32

const (
	//JailFlagCreate use with JailSet to create a jail
	JailFlagCreate = 0x01
	//JailFlagUpdate use with JailSet to update an existing jail
	JailFlagUpdate = 0x02
	//JailFlagAttach use with JailSet to also attach the running process to the jail
	JailFlagAttach = 0x04
	//JailFlagDying allow jails marked as dying
	JailFlagDying = 0x08

	maxhostnamelen = 256
	errormsglen    = 1024
)

var (
	iovErrmsg = []byte("errmsg\000")
)

//JailGet gets values for the parameters in the []syscall.Iovec
func JailGet(iovecs []syscall.Iovec, flags int) (JailID, error) {
	return jailIOVSyscall(syscall.SYS_JAIL_GET, iovecs, flags)
}

//JailSet creates or modifies jails with paramets provided in []syscall.Iovec
func JailSet(iovecs []syscall.Iovec, flags int) (JailID, error) {
	return jailIOVSyscall(syscall.SYS_JAIL_SET, iovecs, flags)
}

//JailAttach attaches the current process to the jail
func JailAttach(jid JailID) error {
	return jailJidSyscall(syscall.SYS_JAIL_ATTACH, jid)
}

//JailRemove destroys the jail, killing all processes in it
func JailRemove(jid JailID) error {
	return jailJidSyscall(syscall.SYS_JAIL_ATTACH, jid)
}

//JailGetName gets the name of the jail associated with JailID
func JailGetName(jid JailID) (string, error) {
	namebuf := make([]byte, maxhostnamelen)

	getparams := make(map[string]interface{})
	getparams["jid"] = jid
	getparams["name"] = namebuf

	iovecs, err := JailParseParametersToIovec(getparams)
	if err != nil {
		return "", err
	}

	_, err = JailGet(iovecs, 0)
	if err != nil {
		return "", err
	}

	return unix.ByteSliceToString(namebuf), nil
}

//JailGetID gets the JailID of jail with the given name
func JailGetID(name string) (JailID, error) {
	getparams := make(map[string]interface{})

	jid, err := strconv.Atoi(name)
	if err == nil {
		if jid == 0 {
			return JailID(0), nil
		}
		getparams["jid"] = int32(jid)
	} else {
		getparams["name"] = name
	}

	iovecs, err := JailParseParametersToIovec(getparams)
	if err != nil {
		return -1, nil
	}

	return JailGet(iovecs, 0)
}

func jailIOVSyscall(callnum uintptr, iovecs []syscall.Iovec, flags int) (JailID, error) {
	errbuf, erriov := makeErrorIov()

	iovecs = append(iovecs, erriov...)

	jid, _, errno := syscall.Syscall(callnum, uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)), uintptr(flags))
	if int32(jid) == -1 || errno != 0 {
		if errbuf[0] == 0 {
			return JailID(jid), errno
		}
		return JailID(jid), errors.New(unix.ByteSliceToString(errbuf))
	}
	return JailID(jid), nil
}

func jailJidSyscall(callnum uintptr, jid JailID) error {
	_, _, errno := syscall.Syscall(callnum, uintptr(jid), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func makeErrorIov() ([]byte, []syscall.Iovec) {
	errmsg := make([]byte, errormsglen)
	erriov := makeJailIovec(iovErrmsg, &errmsg[0], len(errmsg))
	return errmsg, erriov
}
