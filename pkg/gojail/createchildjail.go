package gojail

import (
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

//go:norace
func forkAndCreateChildJail(parentID, iovecs, niovecs, errbufptr uintptr, pipe int) (pid int, err syscall.Errno) {
	var (
		r1      uintptr
		jid     int32
		err1    syscall.Errno
		sendmsg = false
	)

	r1, _, err1 = syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if err1 != 0 {
		return 0, err1
	}

	if r1 != 0 {
		return int(r1), 0
	}

	r1, _, err1 = syscall.RawSyscall(syscall.SYS_JAIL_ATTACH, uintptr(parentID), 0, 0)
	if err1 != 0 {
		goto childerror
	}

	r1, _, err1 = syscall.RawSyscall(syscall.SYS_JAIL_SET, iovecs, niovecs, JailFlagCreate)
	if err1 != 0 || int(r1) == -1 {
		sendmsg = true
		goto childerror
	}
	jid = int32(r1)
	syscall.RawSyscall(syscall.SYS_WRITE, uintptr(pipe), uintptr(unsafe.Pointer(&jid)), unsafe.Sizeof(jid))
	for {
		syscall.RawSyscall(syscall.SYS_EXIT, 0, 0, 0)
	}

childerror:
	syscall.RawSyscall(syscall.SYS_WRITE, uintptr(pipe), uintptr(unsafe.Pointer(&err1)), unsafe.Sizeof(err1))
	if sendmsg {
	    syscall.RawSyscall(syscall.SYS_WRITE, uintptr(pipe), uintptr(errbufptr), errormsglen)
	}
	for {
		syscall.RawSyscall(syscall.SYS_EXIT, 253, 0, 0)
	}
}

func (j *jail) CreateChildJail(parameters map[string]interface{}) (Jail, error) {
	name, err := jailParametersGetName(parameters)
	if err != nil {
		return nil, err
	}

	iovecs, err := JailParseParametersToIovec(parameters)
	if err != nil {
		return nil, err
	}

	err = checkAndIncreaseChildMax(j.jailName)
	if err != nil {
		return nil, err
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	errbuf, erriov := makeErrorIov()

	iovecs = append(iovecs, erriov...)
	syscall.ForkLock.Lock()
	pid, errno := forkAndCreateChildJail(uintptr(j.jailID), uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)), uintptr(unsafe.Pointer(&errbuf[0])), int(writer.Fd()))
	syscall.ForkLock.Unlock()
	if errno != 0 {
		return nil, err
	}
	var waitStatus syscall.WaitStatus
	_, err1 := syscall.Wait4(pid, &waitStatus, 0, nil)
	for err1 == syscall.EINTR {
		_, err1 = syscall.Wait4(pid, &waitStatus, 0, nil)
	}
	if waitStatus.ExitStatus() != 0 {
		errnobuf := make([]byte, 4)
		reader.Read(errnobuf)
		errno := (*syscall.Errno)(unsafe.Pointer(&errnobuf[0]))
		reader.Read(errbuf)
		errstring := unix.ByteSliceToString(errbuf)
		if errstring != "" {
			return nil, errors.New(errstring)
		}
		return nil, error(errno)

	}
	jidbuf := make([]byte, 4)
	reader.Read(jidbuf)
	jid := (*JailID)(unsafe.Pointer(&jidbuf[0]))
	return &jail{
		jailID:   *jid,
		jailName: name,
	}, nil
}

func checkAndIncreaseChildMax(name string) error {
	var childrenMax, childrenCur int32
	getparam := make(map[string]interface{})
	getparam["name"] = name
	getparam["children.max"] = &childrenMax
	getparam["children.cur"] = &childrenCur

	getIovecs, err := JailParseParametersToIovec(getparam)
	if err != nil {
		return err
	}

	_, err = JailGet(getIovecs, 0)
	if err != nil {
		return err
	}

	if childrenCur >= childrenMax {
		setparam := make(map[string]interface{})
		setparam["name"] = name
		setparam["children.max"] = childrenMax + 1

		setIovecs, err := JailParseParametersToIovec(setparam)
		if err != nil {
			return err
		}
		_, err = JailSet(setIovecs, JailFlagUpdate)
		return err
	}
	return nil
}
