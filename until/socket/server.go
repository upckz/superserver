package socket

import (
    "context"
    "fmt"
    "net"
    "strconv"
    "sync"

    "superserver/until/log"

    "github.com/libp2p/go-reuseport"
    "superserver/until/timingwheel"
    "syscall"
    "time"
)

const (
    SocketTypeServer int32 = iota
    SocketTypeClient
)

//Config use for custom the Server
type Config struct {
    IP        string
    Port      int32
    MsgCh     chan *MessageWrapper
    OnConnect OnConnectFunc
    OnClose   OnCloseFunc
    EpollNum  int32
    HbTimeout time.Duration
    Secert    bool
    Ctx       context.Context
}

//Server serve a tcp service
type Server struct {
    conns   *ConnMap //协城安全map
    config  *Config
    name    string
    ctx     context.Context
    cancel  context.CancelFunc
    secert  bool
    epoller []*epoll
    lock    *sync.RWMutex

    listener   net.Listener
    timerWheel *timingwheel.TimingWheel
}

//NewServer create a Server instance with conf
func NewServer(conf *Config) *Server {
    s := &Server{
        config:     conf,
        conns:      NewConnMap(),
        secert:     conf.Secert,
        lock:       &sync.RWMutex{},
        epoller:    make([]*epoll, 0),
        timerWheel: timingwheel.NewTimingWheel(time.Millisecond, 64),
    }
    s.ctx, s.cancel = context.WithCancel(conf.Ctx)
    s.SetName(fmt.Sprint(s))
    go s.timerWheel.Start()
    return s
}

func (s *Server) RemovConn(fd int) {
    s.conns.Remove(fd)
}

func (s *Server) AddConn(fd int, c *TCPConn) {
    s.conns.Put(fd, c)
}

func (s *Server) startEpoll() {

    defer func() {
        if p := recover(); p != nil {
            log.WithFields(log.Fields{}).Panic(p)
        }
        log.WithFields(log.Fields{}).Info("exit startEpoll")
        s.Close()
    }()

    addr := s.config.IP + ":" + strconv.Itoa(int(s.config.Port))

    var err error
    s.listener, err = reuseport.Listen("tcp", addr)
    if err != nil {
        log.WithFields(log.Fields{}).Error(err)
        panic(err)
        return
    }

    epoller, err := MkEpoll()
    if err != nil {
        log.WithFields(log.Fields{}).Error(err)
        panic(err)
        return
    }

    s.insertEpoll(epoller)

    log.WithFields(log.Fields{
        "server": s.name,
        "addr":   s.listener.Addr().String(),
    }).Info("start listen")

    go s.Start(epoller)

    for {
        conn, e := s.listener.Accept()
        if e != nil {
            if ne, ok := e.(net.Error); ok && ne.Temporary() {
                log.WithFields(log.Fields{
                    "err": ne,
                }).Error("accept error")
                continue
            }
            log.WithFields(log.Fields{
                "err": e,
            }).Error("accept error")

            return
        }
        fd := SocketFD(conn)

        sc := NewTCPConn(fd, conn, s, epoller)
        s.AddConn(fd, sc)

        err := epoller.Add(fd)
        if err != nil {
            log.WithFields(log.Fields{
                "err": err,
            }).Error("failed to add connection")
            sc.Close()
        }
        log.WithFields(log.Fields{
            "fd":   fd,
            "addr": conn.RemoteAddr().String(),
        }).Info("New connection incoming")
    }

}

func (s *Server) Start(epoller *epoll) {
    defer func() {
        log.WithFields(log.Fields{}).Debug("exit start")
    }()

    for {
        fds, err := epoller.Wait()
        if err != nil {
            log.WithFields(log.Fields{
                "err": err,
            }).Error("failed to epoll wait")
            continue
        }
        nums := len(fds)
        for i := 0; i < nums; i++ {
            fd := fds[i]
            sc, ok := s.GetConn(fd)
            if ok {
                err = sc.DoRecv()
                if err == nil {
                    epoller.Mode(fd)
                }
            } else {
                log.WithFields(log.Fields{
                    "fd": fd,
                }).Error("not find int server")
            }
        }
        select {
        case <-s.ctx.Done():
            return
        default:
        }
    }
}

//Serve 开始进行监听
func (s *Server) Run() {
    setLimit()

    if s.config.EpollNum < 0 || s.config.EpollNum > 100 {
        s.config.EpollNum = 1
    }
    for i := 0; i < int(s.config.EpollNum); i++ {
        go s.startEpoll()
    }

}

func (s *Server) insertEpoll(epoller *epoll) {
    s.lock.Lock()
    defer s.lock.Unlock()
    s.epoller = append(s.epoller, epoller)
}

//SetName set a name for an instance
func (s *Server) SetName(name string) {
    s.name = name
}

// Unicast unicasts message to a specified conn.
func (s *Server) SendClientMsg(id int, msg *Message) error {
    c, ok := s.conns.Get(id)
    if ok {
        return c.Write(msg)
    }
    return fmt.Errorf("conn %d not found", id)
}

// Broadcast broadcasts message to all server connections managed.
func (s *Server) Broadcast(msg *Message) {

    go func() {
        conns := s.conns.GetAll()
        for _, c := range conns {
            if err := c.Write(msg); err != nil {
                log.WithFields(log.Fields{
                    "err": err,
                }).Error("broadcast error")
            }
        }
    }()

}

// ConnsMap returns connections managed.
func (s *Server) ConnsMap() *ConnMap {
    return s.conns
}

// GetConn returns a server connection with specified ID.
func (s *Server) GetConn(id int) (*TCPConn, bool) {
    return s.conns.Get(id)
}

//Close close the server
func (s *Server) Close() {
    s.cancel()
    conns := s.conns.GetAll()
    for _, conn := range conns {
        conn.Close()
    }
    s.conns.Clear()
    for _, epoller := range s.epoller {
        epoller.Close()
    }
    log.WithFields(log.Fields{
        "name": s.name,
    }).Debug("Server Has Close")

    s.listener.Close()
    s.timerWheel.Stop()

}

func (s *Server) CloseConn(id int) {
    conn, has := s.GetConn(id)
    if has && conn != nil {
        conn.Close()
    }
}

func (s *Server) GetTotalConnect() int {
    return s.conns.Size()
}

func setLimit() {
    var rLimit syscall.Rlimit
    if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
        panic(err)
    }
    rLimit.Cur = rLimit.Max
    if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
        panic(err)
    }
}
