package gojail

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"syscall"
	"unsafe"
)

//JailParseParametersToIovec parses a map[string]interface{} parameter set to []syscall.Iovec
//for use in Jail syscalls requiring []syscall.Iovec
//Byte slices & pointers are considered in/out variables and will be filled with JailGet.
//for setting handing use strings or ints instead.
//gojail uses errmsg, and will error out if it's passed as a key
//No validation is done w.r.t. the type required by the jail parameter
func JailParseParametersToIovec(parameters map[string]interface{}) ([]syscall.Iovec, error) {
	iovecs := make([]syscall.Iovec, 0)
	for key, value := range parameters {
		if key == "errmsg" {
			return nil, errors.New("Usage of errmsg is reserved by gojail")
		}
		parIovec, err := paramToIOVec(key, value)
		if err != nil {
			return nil, err
		}
		//we allow vnet=false to return nil, nil continue in that case
		if parIovec == nil {
			continue
		}
		iovecs = append(iovecs, parIovec...)
	}
	return iovecs, nil
}

func splitParamInPrefixAndParam(p string) (prefix string, param string) {
	s := strings.Split(p, ".")
	param = s[len(s)-1]
	s = s[:len(s)-1]
	prefix = strings.Join(s, ".") + "."
	return prefix, param
}

func paramToIOVec(key string, value interface{}) ([]syscall.Iovec, error) {
	var val *byte

	name, err := syscall.ByteSliceFromString(key)
	if err != nil {
		return nil, err
	}
	valsize := int(4)
	switch v := value.(type) {
	case int32:
		val = (*byte)(unsafe.Pointer(&v))
	case uint32:
		val = (*byte)(unsafe.Pointer(&v))
	case JailID:
		val = (*byte)(unsafe.Pointer(&v))
	case *int32:
		val = (*byte)(unsafe.Pointer(v))
	case *uint32:
		val = (*byte)(unsafe.Pointer(v))
	case *JailID:
		val = (*byte)(unsafe.Pointer(&v))
	case string:
		//Special case: some parameters take strings "disabled", "new" or "inherit"
		//in normal config, but actually map to 0, 1 ,2
		switch v {
		case "disabled":
			return paramToIOVec(key, int32(0))
		case "new":
			return paramToIOVec(key, int32(1))
		case "inherit":
			return paramToIOVec(key, int32(2))
		default:
			val, err = syscall.BytePtrFromString(v)
			if err != nil {
				return nil, err
			}
			valsize = len(v) + 1
		}
	case []byte:
		val = &v[0]
		valsize = len(v)
	case []net.IP:
		//This bit is untested still, use at your own risk
		ipBytes := make([]byte, 0)
		if strings.Contains(key, "ipv4") {
			for _, ip := range v {
				ipv4 := ip.To4()
				if ipv4 != nil {
					ipBytes = append(ipBytes, []byte(ipv4)...)
				} else {
					return nil, fmt.Errorf("could not parse %s to ipv4 address", ip.String())
				}
			}
		} else if strings.Contains(key, "ipv6") {
			for _, ip := range v {
				if ipv4 := ip.To4(); ipv4 == nil {
					ipBytes = append(ipBytes, []byte(ip)...)
				} else {
					return nil, fmt.Errorf("expected ipv6 address got %s", ip.String())
				}
			}
		} else {
			return nil, fmt.Errorf("parsing of net.IP not implemented for key: %s", key)
		}
		val = &ipBytes[0]
		valsize = len(ipBytes)
	case bool:
		//work around for vnet, which is used like a bool, but from testing seems to need
		//an int32 value of 1 when enabling
		if key == "vnet" || key == "novnet" {
			if v {
				return paramToIOVec("vnet", int32(1))
			}
			//change to error if deemed necesarry to have this be consumed by something external
			return nil, nil
		}
		var prefix, param string
		param = key
		//node are deliminated with a "." character, "no" is actually prefixed to the last bit
		//not the entire node
		if strings.Contains(key, ".") {
			prefix, param = splitParamInPrefixAndParam(param)
		}
		if v {
			if strings.HasPrefix(param, "no") {
				param = strings.TrimPrefix(param, "no")
			}
		} else {
			if !strings.HasPrefix(param, "no") {
				param = "no" + param
			}
		}
		var err error
		name, err = syscall.ByteSliceFromString(prefix + param)
		if err != nil {
			return nil, err
		}
		val = nil
		valsize = 0
	default:
		return nil, fmt.Errorf("paramToIOVec: type: %s not implemented", reflect.TypeOf(v))
	}
	return makeJailIovec(name, val, valsize), nil
}

func makeJailIovec(name []byte, value *byte, valuesize int) []syscall.Iovec {
	iovecs := make([]syscall.Iovec, 2)

	iovecs[0].Base = &name[0]
	iovecs[0].SetLen(len(name))

	iovecs[1].Base = value
	iovecs[1].SetLen(valuesize)
	return iovecs
}
