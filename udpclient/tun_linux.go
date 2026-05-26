package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type TunDevice struct {
	name string
	file *os.File
}

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
	IFNAMSIZ  = 16
)

type ifreq struct {
	name [IFNAMSIZ]byte
	flags uint16
}

func NewTunDevice(name string, ip string) (NetDevice, error) {
	fd, err := unix.Open("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var ifr ifreq
	copy(ifr.name[:], name)
	ifr.flags = IFF_TUN | IFF_NO_PI

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		unix.Close(fd)
		return nil, fmt.Errorf("ioctl TUNSETIFF failed: %v", errno)
	}

	file := os.NewFile(uintptr(fd), "/dev/net/tun")
	tunName := string(ifr.name[:])

	addr, _, err := net.ParseCIDR(ip + "/24")
	if err != nil {
		return nil, err
	}

	if err := setTunIP(tunName, addr); err != nil {
		return nil, err
	}

	if err := setTunUp(tunName); err != nil {
		return nil, err
	}

	return &TunDevice{
		name: tunName,
		file: file,
	}, nil
}

func setTunIP(name string, addr net.IP) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	type ifreqAddr struct {
		name [IFNAMSIZ]byte
		addr unix.RawSockaddrInet4
	}

	var ifr ifreqAddr
	copy(ifr.name[:], name)
	ifr.addr.Family = unix.AF_INET
	copy(ifr.addr.Addr[:], addr.To4())

	SIOCSIFADDR := 0x8916
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(SIOCSIFADDR), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return fmt.Errorf("ioctl SIOCSIFADDR failed: %v", errno)
	}

	return nil
}

func setTunUp(name string) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var ifr ifreq
	copy(ifr.name[:], name)

	SIOCGIFFLAGS := 0x8913
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(SIOCGIFFLAGS), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return fmt.Errorf("ioctl SIOCGIFFLAGS failed: %v", errno)
	}

	ifr.flags |= uint16(syscall.IFF_UP | syscall.IFF_RUNNING)

	SIOCSIFFLAGS := 0x8914
	_, _, errno = unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(SIOCSIFFLAGS), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return fmt.Errorf("ioctl SIOCSIFFLAGS failed: %v", errno)
	}

	return nil
}

func (t *TunDevice) Read(packet []byte) (int, error) {
	return t.file.Read(packet)
}

func (t *TunDevice) Write(packet []byte) (int, error) {
	return t.file.Write(packet)
}

func (t *TunDevice) Close() error {
	return t.file.Close()
}
