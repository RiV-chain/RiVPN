//go:build !mobile
// +build !mobile

package tun

// The darwin platform specific tun parts

import (
	"encoding/binary"
	"golang.org/x/sys/unix"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"unsafe"

	wgtun "golang.zx2c4.com/wireguard/tun"
)

// Configures the "utun" adapter with the correct IPv6 address and MTU.
func (tun *TunAdapter) setup(ifname string, addr string, mtu uint64) error {
	if ifname == "auto" {
		ifname = "utun"
	}
	iface, err := wgtun.CreateTUN(ifname, int(mtu))
	if err != nil {
		panic(err)
	}
	tun.iface = iface
	if m, err := iface.MTU(); err == nil {
		tun.mtu = getSupportedMTU(uint64(m))
	} else {
		tun.mtu = 0
	}
	return tun.setupAddress(addr)
}

const (
	darwin_SIOCAIFADDR_IN6       = 2155899162 // netinet6/in6_var.h
	darwin_IN6_IFF_NODAD         = 0x0020     // netinet6/in6_var.h
	darwin_IN6_IFF_SECURED       = 0x0400     // netinet6/in6_var.h
	darwin_ND6_INFINITE_LIFETIME = 0xFFFFFFFF // netinet6/nd6.h
)

// nolint:structcheck
type in6_addrlifetime struct {
	ia6t_expire    float64 // nolint:unused
	ia6t_preferred float64 // nolint:unused
	ia6t_vltime    uint32
	ia6t_pltime    uint32
}

// nolint:structcheck
type sockaddr_in6 struct {
	sin6_len      uint8
	sin6_family   uint8
	sin6_port     uint8  // nolint:unused
	sin6_flowinfo uint32 // nolint:unused
	sin6_addr     [8]uint16
	sin6_scope_id uint32 // nolint:unused
}

// nolint:structcheck
type in6_aliasreq struct {
	ifra_name       [16]byte
	ifra_addr       sockaddr_in6
	ifra_dstaddr    sockaddr_in6 // nolint:unused
	ifra_prefixmask sockaddr_in6
	ifra_flags      uint32
	ifra_lifetime   in6_addrlifetime
}

type ifreq struct {
	ifr_name [16]byte
	ifru_mtu uint32
}

// struct ifalias_req
type aliasreq struct {
	ifra_name    [unix.IFNAMSIZ]byte
	ifra_addr    unix.RawSockaddrInet4
	ifra_dstaddr unix.RawSockaddrInet4
	ifra_mask    unix.RawSockaddrInet4
}

// Implementation: Adds an IPv4 address to an interface.
func addressAdd4(intf_name string, ipv4 []byte) error {

	var fd int
	var err error

	ip := [4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]}
	dst_ip := [4]byte{0, 0, 0, 0}
	// First ------------------------------------------------------------------
	//	Open an AF_INET Socket
	// ------------------------------------------------------------------------
	fd, err = unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	var ifra_name [unix.IFNAMSIZ]byte
	copy(ifra_name[:], tun.Name())
	// Second -----------------------------------------------------------------
	//	Prepare the ioctl Request Argument
	// ------------------------------------------------------------------------
	ifra4 := aliasreq{
		ifra_name: ifra_name,
		ifra_addr: unix.RawSockaddrInet4{
			Len:    unix.SizeofSockaddrInet4,
			Family: unix.AF_INET,
			Addr:   ip,
		},
		ifra_dstaddr: unix.RawSockaddrInet4{
			Len:    unix.SizeofSockaddrInet4,
			Family: unix.AF_INET,
			Addr:   dst_ip,
		},
		ifra_mask: unix.RawSockaddrInet4{
			Len:    unix.SizeofSockaddrInet4,
			Family: unix.AF_INET,
			Addr:   netip.MustParseAddr(net.IP(net.CIDRMask(8, 32)).String()).As4(),
		},
	}

	// Third ------------------------------------------------------------------
	//	Call ioctl to set the Address
	// ------------------------------------------------------------------------
	return ioctl(fd, unix.SIOCAIFADDR, uintptr(unsafe.Pointer(&ifra4)))
}

// Sets the IPv6 address of the utun adapter. On Darwin/macOS this is done using
// a system socket and making direct syscalls to the kernel.
func (tun *TunAdapter) setupAddress(addr string) error {
	var fd int
	var err error

	if fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, 0); err != nil {
		tun.log.Printf("Create AF_SYSTEM socket failed: %v.", err)
		return err
	}

	var ar in6_aliasreq
	copy(ar.ifra_name[:], tun.Name())

	ar.ifra_prefixmask.sin6_len = uint8(unsafe.Sizeof(ar.ifra_prefixmask))
	b := make([]byte, 16)
	binary.LittleEndian.PutUint16(b, uint16(0xFE00))
	ar.ifra_prefixmask.sin6_addr[0] = binary.BigEndian.Uint16(b)

	ar.ifra_addr.sin6_len = uint8(unsafe.Sizeof(ar.ifra_addr))
	ar.ifra_addr.sin6_family = unix.AF_INET6
	parts := strings.Split(strings.Split(addr, "/")[0], ":")
	for i := 0; i < 8; i++ {
		addr, _ := strconv.ParseUint(parts[i], 16, 16)
		b := make([]byte, 16)
		binary.LittleEndian.PutUint16(b, uint16(addr))
		ar.ifra_addr.sin6_addr[i] = binary.BigEndian.Uint16(b)
	}

	ar.ifra_flags |= darwin_IN6_IFF_NODAD
	ar.ifra_flags |= darwin_IN6_IFF_SECURED

	ar.ifra_lifetime.ia6t_vltime = darwin_ND6_INFINITE_LIFETIME
	ar.ifra_lifetime.ia6t_pltime = darwin_ND6_INFINITE_LIFETIME

	var ir ifreq
	copy(ir.ifr_name[:], tun.Name())
	ir.ifru_mtu = uint32(tun.mtu)

	tun.log.Infof("Interface name: %s", ar.ifra_name)
	tun.log.Infof("Interface IPv6: %s", addr)
	tun.log.Infof("Interface MTU: %d", ir.ifru_mtu)

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(darwin_SIOCAIFADDR_IN6), uintptr(unsafe.Pointer(&ar))); errno != 0 {
		err = errno
		tun.log.Errorf("Error in darwin_SIOCAIFADDR_IN6: %v", errno)
		return err
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.SIOCSIFMTU), uintptr(unsafe.Pointer(&ir))); errno != 0 {
		err = errno
		tun.log.Errorf("Error in SIOCSIFMTU: %v", errno)
		return err
	}

	if address, errno := netip.ParsePrefix(addr); errno == nil {
		ipv6 := address.Addr().Unmap().AsSlice()
		ipv6[0] = 10
		ipv6[3] = ipv6[3]>>1 + 1
		addressAdd4(tun.Name(), ipv6[:4])
	} else {
		err = errno
		tun.log.Errorf("Could not map IPv4 address from IPv6: %v", errno)
		return err
	}
	return nil
}

// Syscall wrapper for calling ioctl requests
func ioctl(fd int, request int, argp uintptr) error {
	_, _, errorp := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(request), argp)
	return errorp
}
