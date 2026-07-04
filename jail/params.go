package jail

import (
	"fmt"
	"net/netip"
	"syscall"
)

// CreateParams is a limited subset of the parameters available in jail.conf(5) for use with jail(8).
type CreateParams struct {
	Name       string
	Root       string
	Hostname   string
	Domainname string
	Host       string
	IP4        string
	IP4Addr    []string
	IP6        string
	IP6Addr    []string
	VNet       string
	// VNetInterface
	// Deprecated: not used
	VNetInterface []string
	// EnforceStatfs controls mount visibility (0, 1, or 2); nil leaves the
	// kernel default.
	EnforceStatfs *int
}

func (c *CreateParams) iovec() ([]syscall.Iovec, error) {
	iovec := make([]syscall.Iovec, 0)

	name, err := stringIovec("name", c.Name)
	if err != nil {
		return nil, err
	}
	iovec = append(iovec, name...)

	root, err := stringIovec("path", c.Root)
	if err != nil {
		return nil, err
	}
	iovec = append(iovec, root...)

	if c.Hostname != "" {
		hostname, err := stringIovec("host.hostname", c.Hostname)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, hostname...)
	}

	// host.domainname sets the jail's YP/NIS domain.  Like host.hostname, it
	// makes the kernel give the jail its own UTS information (host=new,
	// PR_HOST), so the value is private to the jail.
	if c.Domainname != "" {
		domainname, err := stringIovec("host.domainname", c.Domainname)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, domainname...)
	}

	if c.Host != "" {
		var host int32
		switch c.Host {
		case "new":
			host = 1
		case "inherit":
			if c.Hostname != "" {
				return nil, fmt.Errorf("jail: validation failure: cannot set Hostname %q with Host mode %q", c.Hostname, c.Host)
			}
			if c.Domainname != "" {
				return nil, fmt.Errorf("jail: validation failure: cannot set Domainname %q with Host mode %q", c.Domainname, c.Host)
			}
			host = 2
		default:
			return nil, fmt.Errorf("jail: unknown Host type %q", c.Host)
		}
		hostio, err := int32Iovec("host", host)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, hostio...)
	}

	if c.VNet != "" {
		var vnet int32
		switch c.VNet {
		case "new":
			vnet = 1
		case "inherit":
			vnet = 2
		default:
			return nil, fmt.Errorf("jail: unknown VNet type %q", c.VNet)
		}
		vnetio, err := int32Iovec("vnet", vnet)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, vnetio...)
	}

	if c.IP4 != "" {
		var ip4 int32
		switch c.IP4 {
		case "disable":
			ip4 = 0
		case "new":
			ip4 = 1
		case "inherit":
			ip4 = 2
		default:
			return nil, fmt.Errorf("jail: unknown IP4 type %q", c.IP4)
		}
		ip4io, err := int32Iovec("ip4", ip4)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, ip4io...)
	}

	if len(c.IP4Addr) > 0 {
		ip4Addrs := make([]netip.Addr, 0)
		for _, addr := range c.IP4Addr {
			ip4Addr, err := netip.ParseAddr(addr)
			if err != nil {
				return nil, fmt.Errorf("jail: failed to parse %q as IPv4: %w", addr, err)
			}
			if !ip4Addr.Is4() {
				return nil, fmt.Errorf("jail: invalid IP4 address %q", c.IP4Addr)
			}
			ip4Addrs = append(ip4Addrs, ip4Addr)
		}
		ip4Addrio, err := netIPIovec("ip4.addr", ip4Addrs)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, ip4Addrio...)
	}

	if c.IP6 != "" {
		var ip6 int32
		switch c.IP6 {
		case "disable":
			ip6 = 0
		case "new":
			ip6 = 1
		case "inherit":
			ip6 = 2
		default:
			return nil, fmt.Errorf("jail: unknown IP6 type %q", c.IP6)
		}
		ip6io, err := int32Iovec("ip6", ip6)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, ip6io...)
	}

	if len(c.IP6Addr) > 0 {
		ip6Addrs := make([]netip.Addr, 0)
		for _, addr := range c.IP6Addr {
			ip6Addr, err := netip.ParseAddr(addr)
			if err != nil {
				return nil, fmt.Errorf("jail: failed to parse %q as IPv6: %w", addr, err)
			}
			if !ip6Addr.Is6() || ip6Addr.Is4In6() {
				return nil, fmt.Errorf("jail: invalid IP6 address %q", addr)
			}
			ip6Addrs = append(ip6Addrs, ip6Addr)
		}
		ip6Addrio, err := netIPIovec("ip6.addr", ip6Addrs)
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, ip6Addrio...)
	}

	if c.EnforceStatfs != nil {
		v := *c.EnforceStatfs
		if v < 0 || v > 2 {
			return nil, fmt.Errorf("jail: invalid enforce_statfs value %d (must be 0, 1, or 2)", v)
		}
		esio, err := int32Iovec("enforce_statfs", int32(v))
		if err != nil {
			return nil, err
		}
		iovec = append(iovec, esio...)
	}

	persist, err := nilIovec("persist")
	if err != nil {
		return nil, err
	}
	iovec = append(iovec, persist...)

	return iovec, nil
}
