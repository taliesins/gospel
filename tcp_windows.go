package gospel

import (
	"net"
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	ws2_32 = syscall.NewLazyDLL("ws2_32.dll")
	procAccept = ws2_32.NewProc("accept")
)

type Listener struct {
	fd uintptr
	sa syscall.Sockaddr
	addr *net.IPAddr
}

type Conn struct {
	fd uintptr
	addr *net.TCPAddr
}

func Listen(protocol string, address string) (*Listener, error) {
	tcpAddress, err := net.ResolveTCPAddr(protocol, address)
	if err != nil {
		return nil, err
	}

	socketHandle, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	var socketAddress syscall.Sockaddr
	switch protocol {
	case "tcp", "tcp4":
		socketAddressIpV4 := new(syscall.SockaddrInet4)
		socketAddressIpV4.Port = tcpAddress.Port
		copy(socketAddressIpV4.Addr[:], tcpAddress.IP.To4())
		socketAddress = socketAddressIpV4
	case "tcp6":
		socketAddressIpv6 := new(syscall.SockaddrInet6)
		socketAddressIpv6.Port = tcpAddress.Port
		copy(socketAddressIpv6.Addr[:], tcpAddress.IP.To16())
		socketAddress = socketAddressIpv6
	}

	err = syscall.Bind(socketHandle, socketAddress)
	if err != nil {
		return nil, err
	}

	err = syscall.Listen(socketHandle, syscall.SOMAXCONN)
	if err != nil {
		return nil, err
	}

	ssa, err := syscall.Getsockname(syscall.Handle(socketHandle))
	if err != nil {
		return nil, err
	}
	ta := &net.IPAddr{IP: ssa.(*syscall.SockaddrInet4).Addr[0:]}
	return &Listener{uintptr(socketHandle), socketAddress, ta}, nil
}

func (l *Listener) Addr() net.Addr {
	return l.addr

}

func (l *Listener) Close() error {
	return syscall.Closesocket(syscall.Handle(l.fd))
}

func (l *Listener) Accept() (net.Conn, error) {
	var sa syscall.SockaddrInet4
	sl := unsafe.Sizeof(sa)
	newfd, r1, err := procAccept.Call(uintptr(l.fd), uintptr(unsafe.Pointer(&sa)), uintptr(unsafe.Pointer(&sl)))
	if err != nil && r1 == 0 {
		return nil, err
	}
	//return &Conn{uintptr(newfd), nil}, nil
	return net.FileConn(os.NewFile(newfd, "sysfile"))
}

func (c *Conn) Read(b []byte) (n int, e error) {
	var buf syscall.WSABuf
	buf.Buf = &b[0]
	buf.Len = uint32(len(b))
	var qty, flags uint32
	err := syscall.WSARecv(syscall.Handle(c.fd), &buf, 1, &qty, &flags, nil, nil)
	return int(qty), err
}

func (c *Conn) Write(b []byte) (int, error) {
	return syscall.Write(syscall.Handle(c.fd), b)
}

func (c *Conn) Close() error {
	return syscall.Closesocket(syscall.Handle(c.fd))
}

func (c *Conn) LocalAddr() net.Addr {
	return c.addr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.addr
}

func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}

