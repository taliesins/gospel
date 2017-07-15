package gospel

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	modws2_32   = syscall.NewLazyDLL("ws2_32.dll")
	procWSADuplicateSocketW  = modws2_32.NewProc("WSADuplicateSocketW")
	procWSASocketW           = modws2_32.NewProc("WSASocketW")
)

func Syscall(trap uintptr, args ...uintptr) (r1, r2 uintptr, err syscall.Errno) {
	switch len(args) {
	case 0: return syscall.Syscall(trap, 0, 0, 0, 0)
	case 1: return syscall.Syscall(trap, 1, args[0], 0, 0)
	case 2: return syscall.Syscall(trap, 2, args[0], args[1], 0)
	case 3: return syscall.Syscall(trap, 3, args[0], args[1], args[2])
	case 4: return syscall.Syscall6(trap, 4, args[0], args[1], args[2], args[3],
		0, 0)
	case 5: return syscall.Syscall6(trap, 5, args[0], args[1], args[2], args[3],
		args[4], 0)
	case 6: return syscall.Syscall6(trap, 6, args[0], args[1], args[2], args[3],
		args[4], args[5])
	case 7: return syscall.Syscall9(trap, 7, args[0], args[1], args[2], args[3],
		args[4], args[5], args[6], 0, 0)
	case 8: return syscall.Syscall9(trap, 8, args[0], args[1], args[2], args[3],
		args[4], args[5], args[6], args[7], 0)
	case 9: return syscall.Syscall9(trap, 9, args[0], args[1], args[2], args[3],
		args[4], args[5], args[6], args[7], args[8])
	case 10: return syscall.Syscall12(trap, 10, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], 0, 0)
	case 11: return syscall.Syscall12(trap, 11, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], args[10], 0)
	case 12: return syscall.Syscall12(trap, 12, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], args[10],
		args[11])
	case 13: return syscall.Syscall15(trap, 13, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], args[10],
		args[11], args[12],0,0)
	case 14: return syscall.Syscall15(trap, 14, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], args[10],
		args[11], args[12], args[13],0)
	case 15: return syscall.Syscall15(trap, 15, args[0], args[1], args[2],
		args[3], args[4], args[5], args[6], args[7], args[8], args[8], args[10],
		args[11], args[12],args[13],args[14])
	}
	panic("too many args")
}

func WSADuplicateSocket(handle syscall.Handle, pid uint32, pi *syscall.WSAProtocolInfo) (sockerr error) {
	r0, _, _ := Syscall(procWSADuplicateSocketW.Addr(), uintptr(handle), uintptr(pid), uintptr(unsafe.Pointer(pi)))
	if r0 != 0 {
		sockerr = syscall.Errno(r0)
	}
	return
}

func WSASocket(af int32, stype int32, protocol int32, pi *syscall.WSAProtocolInfo, g uint32, flags uint32) (handle syscall.Handle, err error) {
	r0, _, e1 := Syscall(procWSASocketW.Addr(), uintptr(af), uintptr(stype), uintptr(protocol), uintptr(unsafe.Pointer(pi)), uintptr(g), uintptr(flags))
	handle = syscall.Handle(r0)
	if handle == syscall.InvalidHandle {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func (g *Gospel) exec(listener net.Listener) (*os.Process, error) {
	fd := sysfd(listener)

	socketFile, err := ioutil.TempFile(os.TempDir(), "gospel-fd")
	if err != nil {
		return nil, err
	}
	os.Setenv("GOSPEL_FD", socketFile.Name())
	err = g.cmd.Start()
	if err != nil {
		socketFile.Close()
		os.Remove(socketFile.Name())
		return nil, err
	}
	b := make([]byte, int(unsafe.Sizeof(syscall.WSAProtocolInfo{})))
	err = WSADuplicateSocket(syscall.Handle(fd), uint32(g.cmd.Process.Pid), (*syscall.WSAProtocolInfo)(unsafe.Pointer(&b[0])))
	if err != nil {
		socketFile.Close()
		os.Remove(socketFile.Name())
		return nil, err
	}
	socketFile.Write(b)
	socketFile.Close()
	return g.cmd.Process, err
}

func ListenerFromEnv() (net.Listener, error) {
	socketFileName := os.Getenv("GOSPEL_FD")
	l := int(unsafe.Sizeof(syscall.WSAProtocolInfo{}))
	var socketFileContents []byte
	var err error
	for n := 0; n < 3; n++ {
		socketFileContents, err = ioutil.ReadFile(socketFileName)
		if len(socketFileContents) == l {
			break
		}
		time.Sleep(1e9)
	}
	if len(socketFileContents) == 0 {
		return nil, errors.New("server not found")
	}
	if err != nil {
		return nil, err
	}
	pi := (*syscall.WSAProtocolInfo)(unsafe.Pointer(&socketFileContents[0]))
	fd, err := WSASocket(-1, -1, -1, pi, 0, 0)
	if err != nil {
		return nil, err
	}

	/*
	syscall.SetNonblock(syscall.Handle(fd), true)

	sa, err := syscall.Getsockname(syscall.Handle(fd))
	if err != nil {
		return nil, err
	}
	ta := &net.IPAddr{IP: sa.(*syscall.SockaddrInet4).Addr[0:]}
	return &Listener{uintptr(fd), sa, ta}, nil
	*/
	return net.FileListener(os.NewFile(uintptr(fd), "sysfile"))
}
