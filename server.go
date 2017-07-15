package gospel

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

type Gospel struct {
	cmd    *exec.Cmd
}

func New(cmd *exec.Cmd) *Gospel {
	return &Gospel{cmd}
}

func (g *Gospel) Listen(address string) error {
	var listener net.Listener
	var err error
	if runtime.GOOS == "windows" {
		listener, err = Listen("tcp", address)
	} else {
		listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		return err
	}

	process, err := g.exec(listener)
	if err != nil {
		return err
	}
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP)
	for {
		switch sig := <-c; sig {
		case syscall.SIGHUP:
			child, err := g.exec(listener)
			if err != nil {
				return err
			}
			process.Signal(syscall.SIGINT)
			process.Wait()
			process = child
		case syscall.SIGINT:
			signal.Stop(c)
			listener.Close()
			process.Signal(syscall.SIGINT)
			_, err := process.Wait()
			return err
		}
	}
}

func sysfd(listener net.Listener) uintptr {
	if ll, ok := listener.(*Listener); ok {
		return ll.fd
	}
	return *(*uintptr)(unsafe.Pointer(reflect.ValueOf(listener).Elem().FieldByName("fd").Elem().FieldByName("sysfd").Addr().Pointer()))
}

func fd() (uintptr, error) {
	socketFileName := os.Getenv("GOSPEL_FD")
	if socketFileName == "" {
		return 0, errors.New("server not found")
	}
	fd, err := strconv.Atoi(socketFileName)
	if err != nil {
		return 0, err
	}
	return uintptr(fd), nil
}
