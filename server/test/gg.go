package main 
import (
  "golang.org/x/sys/unix"
  "sync/atomic"
  "net"
  "fmt"
  "github.com/libp2p/go-reuseport"
    "os"
    "os/signal"
    "syscall"
      "reflect"

)

const (
  POLL_OPENED uint32 = iota
  POLL_CLOSED
)

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


type poll struct {
  fd     int
  status uint32
}

func mkPoll() (*poll, error) {
  fd, err := unix.EpollCreate1(0)
  if err != nil {
    return nil, err
  }
  return &poll{
    fd: fd,
  }, nil
}

func (p *poll) triggerClose() {
  atomic.AddUint32(&p.status, POLL_CLOSED)
}

func (p *poll) close() error {
  return unix.Close(p.fd)
}

func (p *poll) Wait(f func(fd int, mode int32) error) error {

  events := make([]unix.EpollEvent, 64)
  for {
    n, err := unix.EpollWait(p.fd, events, 64)
    if err != nil && err != unix.EINTR {
      return err
    }
  // fmt.Printf("n....[%d]\n", n)
   
    for i := 0; i < n; i++ {
      var mode int32
      if events[i].Events&(unix.EPOLLIN|unix.EPOLLRDHUP|unix.EPOLLHUP|unix.EPOLLERR) != 0 {
        mode += 'r'
      }
      if events[i].Events&(unix.EPOLLOUT|unix.EPOLLHUP|unix.EPOLLERR) != 0 {
        mode += 'w'
      }
      if mode != 0 {
        if fd := int(events[i].Fd); fd != 0 {
          if err := f(fd, mode); err != nil {
            return err
          }
        }
      }
    }
  }
}

func (p *poll) addLn(fd int) {
  if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd,
    &unix.EpollEvent{Fd: int32(fd),
      Events: unix.EPOLLIN | unix.EPOLLET,
    },
  ); err != nil {
    panic(err)
  }
}

func (p *poll) addFd(fd int) {
  if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd,
    &unix.EpollEvent{Fd: int32(fd),
      Events: unix.EPOLLIN | unix.EPOLLOUT | unix.EPOLLPRI | unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLET,
    },
  ); err != nil {
    panic(err)
  }
}

func (p *poll) remove(fd int) {
  if err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil); err != nil {
    panic(err)
  }
}


type Server struct {
    connManager map[int]int
    listen        net.Listener
    epoller     *poll
}

func NewServer() *Server {
    s := &Server {
        connManager: make(map[int]int),
    }
    return s
}

func (s *Server) Start() error {

  var err error
  s.listen, err = reuseport.Listen("tcp", ":9100")
    if err != nil {
        fmt.Printf("Error:%v\n", err)

        return err
  }

  s.epoller, err = mkPoll()
    if err != nil {
        fmt.Printf("Error:%v\n", err)
        return err
  }
   tcpListener :=  s.listen.(*net.TCPListener)
  lnFile, err := tcpListener.File()
  if err != nil {
   return err
  }
 s.epoller.addLn(int(lnFile.Fd()))
  go s.run()
  return nil
  
}


func (s *Server)run() {

    s.epoller.Wait(func(fd int, mode int32) error {
      _, ok := s.connManager[fd]
     // fmt.Printf("oooooooo.......fd[%d]\n", fd)
      if !ok  {
        return s.accept(fd)
      }
      //fmt.Printf("fd ..%d\n", fd)
      if mode == 'r' || mode == 'r'+'w' {
          fmt.Println("read ...")
      }
      if mode == 'w' || mode == 'r'+'w' {
         fmt.Println("write ...")
      }
      return nil
    })

}

func (s *Server)accept(fd int) error{
    for {

       // conn, e := s.listen.Accept()
       //  if e != nil {
       //      if ne, ok := e.(net.Error); ok && ne.Temporary() {
       //           fmt.Printf("accept temp err: %v\n", ne)
       //          return e
       //      }

       //       fmt.Printf("accept err: %v\n", e)
       //      return e
       //  }
       //  nfd := socketFD(conn)

       //   fmt.Printf("New connection connId[%d] incoming <%v>\n", nfd, conn.RemoteAddr())

      nfd, _, err := unix.Accept(fd)
      if err != nil {
        if err == unix.EAGAIN {
          return nil
        }
        return err
      }
      if err := unix.SetNonblock(nfd, true); err != nil {
        return err
      }
      
      _, ok := s.connManager[fd] 
      if ok {
        fmt.Printf("ERROR .............\n")
        continue
      }
      s.connManager[nfd] = 1
      s.epoller.addFd(nfd)
      fmt.Printf("New connection connId[%d] incoming[%d] \n", nfd,len(s.connManager))
    }
    return nil
}

func main() {

  //s := NewServer()
 // s.Start()

 //WaitSignalSynchronized()
 //
 

 ln, err := reuseport.Listen("tcp", ":8972")
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
          log.Printf("failed to epoll wait %v", err)
          continue
        }
        for _, conn := range connections {
          if conn == nil {
            break
          }
          io.CopyN(conn, conn, 8)
          if err != nil {
            if err := epoller.Remove(conn); err != nil {
              log.Printf("failed to remove %v", err)
            }
            conn.Close()
          }

          opsRate.Mark(1)
        }
  }

  }(epoller)

  for {
    conn, e := ln.Accept()
    if e != nil {
      if ne, ok := e.(net.Error); ok && ne.Temporary() {
        log.Printf("accept temp err: %v", ne)
        continue
      }

      log.Printf("accept err: %v", e)
      return
    }

    if err := epoller.Add(conn); err != nil {
      log.Printf("failed to add connection %v", err)
      conn.Close()
    }
  }

}