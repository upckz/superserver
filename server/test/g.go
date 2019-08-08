
package main 
import (
  "golang.org/x/sys/unix"
  "sync"
  "net"
  "fmt"
  "github.com/libp2p/go-reuseport"
    "os"
    "os/signal"
    "syscall"
      "reflect"

    "log"

)
//WaitSignalSynchronized 同步等待系统信号并处理
func WaitSignalSynchronized() {
    sigsCh := make(chan os.Signal, 1)
    doneCh := make(chan struct{})

    signal.Notify(sigsCh, syscall.SIGABRT, syscall.SIGPIPE, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.Signal(10))

    go func() {
        for {
            select {
            // case <-time.After(30 * time.Second):
            //  log.Infoln("Time Out Close")
            //  close(doneCh)
            //  return
            case sig := <-sigsCh:
                fmt.Printf("Receive Signal: ", sig)
                switch sig {
                case syscall.Signal(10):
                    close(doneCh)
                    return
                case syscall.SIGINT:
                    close(doneCh)
                    return
                default:
                    continue
                }
            }
        }
    }()
    fmt.Printf("Waiting For Singals\n")
    <-doneCh
    fmt.Printf("Need Exit\n")
}


type epoll struct {
  fd          int
  connections map[int]net.Conn
  lock        *sync.RWMutex
}

func MkEpoll() (*epoll, error) {
  fd, err := unix.EpollCreate1(0)
  if err != nil {
    return nil, err
  }
  return &epoll{
    fd:          fd,
    lock:        &sync.RWMutex{},
    connections: make(map[int]net.Conn),
  }, nil
}

func (e *epoll) Add(conn net.Conn) error {
  // Extract file descriptor associated with the connection
  fd := socketFD(conn)
  err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
  if err != nil {
    return err
  }
  e.lock.Lock()
  defer e.lock.Unlock()
  e.connections[fd] = conn
  if len(e.connections)%100 == 0 {
    log.Printf("total number of connections: %v", len(e.connections))
  }
  return nil
}

func (e *epoll) Remove(conn net.Conn) error {
  fd := socketFD(conn)
  err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
  if err != nil {
    return err
  }
  e.lock.Lock()
  defer e.lock.Unlock()
  delete(e.connections, fd)
  if len(e.connections)%100 == 0 {
    log.Printf("total number of connections: %v", len(e.connections))
  }
  return nil
}

func (e *epoll) Wait() ([]net.Conn, error) {
  events := make([]unix.EpollEvent, 100)
  n, err := unix.EpollWait(e.fd, events, 100)
  if err != nil {
    return nil, err
  }
  e.lock.RLock()
  defer e.lock.RUnlock()
  var connections []net.Conn
  for i := 0; i < n; i++ {
    conn := e.connections[int(events[i].Fd)]
    connections = append(connections, conn)
  }
  return connections, nil
}

func socketFD(conn net.Conn) int {
  //tls := reflect.TypeOf(conn.UnderlyingConn()) == reflect.TypeOf(&tls.Conn{})
  // Extract the file descriptor associated with the connection
  //connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
  tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
  //if tls {
  //  tcpConn = reflect.Indirect(tcpConn.Elem())
  //}
  fdVal := tcpConn.FieldByName("fd")
  pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

  return int(pfdVal.FieldByName("Sysfd").Int())
}


func main() {

  //s := NewServer()
 // s.Start()

 //WaitSignalSynchronized()
 //
 
 connManager := make(map[int]int)


 ln, err := reuseport.Listen("tcp", ":9100")
  if err != nil {
    panic(err)
  }

  epoller, err := MkEpoll()
  if err != nil {
    panic(err)
  }

  go func(epoller *epoll){
    for {
        connections, err := epoller.Wait()
        if err != nil {
          log.Printf("failed to epoll wait %v\n", err)
          continue
        }
        for _, conn := range connections {
          if conn == nil {
            break
          }
          
          if err != nil {
            if err := epoller.Remove(conn); err != nil {
              log.Printf("failed to remove %v\n", err)
            }
            conn.Close()
          }

          
        }
  }

  }(epoller)

  cnt := 0
  for {
    conn, e := ln.Accept()
    if e != nil {
      if ne, ok := e.(net.Error); ok && ne.Temporary() {
        log.Printf("accept temp err: %v\n", ne)
        continue
      }

      log.Printf("accept err: %v\n", e)
      return
    }
    nfd := socketFD(conn)
    connManager[nfd] = 1
    cnt++

    fmt.Printf("total [%d]\n", len(connManager))

    if err := epoller.Add(conn); err != nil {
      log.Printf("failed to add connection %v\n", err)
      conn.Close()
    }
  }

}