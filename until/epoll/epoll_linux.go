package epoll

import (
	"golang.org/x/sys/unix"
	"net"
	"reflect"
	"sync"
	"syscall"
)

type Epoll struct {
	fd   int
	lock *sync.RWMutex
}

func MkEpoll() (*Epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &Epoll{
		fd:   fd,
		lock: &sync.RWMutex{},
	}, nil
}

func (e *Epoll) Close() error {
	return unix.Close(e.fd)
}

func (e *Epoll) Add(fd int) error {

	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}
	// Extract file descriptor associated with the connection
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	return nil
}

func (e *Epoll) Mode(fd int) error {

	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	return nil
}

func (e *Epoll) Remove(fd int) error {
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	return nil
}

func (e *Epoll) Wait() ([]int, error) {

	var fds []int

	events := make([]unix.EpollEvent, 256)
	n, err := unix.EpollWait(e.fd, events, 256)
	if err != nil {
		return fds, err
	}

	for i := 0; i < n; i++ {
		ev := &events[i]
		if ev.Events == 0 {
			continue
		}
		fds = append(fds, int(events[i].Fd))

	}
	return fds, nil
}

func SocketFD(conn net.Conn) int {
	//tls := reflect.TypeOf(conn.UnderlyingConn()) == reflect.TypeOf(&tls.Conn{})
	// Extract the file descriptor associated with the connection
	//connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	//if tls {
	//	tcpConn = reflect.Indirect(tcpConn.Elem())
	//}
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}
