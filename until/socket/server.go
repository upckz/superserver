package socket

import (
    "context"
    "fmt"
    "net"
    "strconv"
    "sync"

    "superserver/until/common"
    log "superserver/until/czlog"

    "github.com/libp2p/go-reuseport"
    "syscall"
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
    HbTimeout int32
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

    listener net.Listener
}

//NewServer create a Server instance with conf
func NewServer(conf *Config) *Server {
    s := &Server{
        config:  conf,
        conns:   NewConnMap(),
        secert:  conf.Secert,
        lock:    &sync.RWMutex{},
        epoller: make([]*epoll, 0),
    }
    s.ctx, s.cancel = context.WithCancel(conf.Ctx)
    s.SetName(fmt.Sprint(s))
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
            log.Errorln(common.GlobalPanicLog(p))
        }
        log.Debugf("exit startEpoll")
    }()

    addr := s.config.IP + ":" + strconv.Itoa(int(s.config.Port))

    var err error
    s.listener, err = reuseport.Listen("tcp", addr)
    if err != nil {
        log.Errorf("Error:%v", err)
        panic(err)
        return
    }

    epoller, err := MkEpoll()
    if err != nil {
        log.Errorf("Error:%v", err)
        panic(err)
        return
    }

    s.insertEpoll(epoller)

    log.Infof("Sever[%s] start listen on: %v", s.name, s.listener.Addr())

    go s.Start(epoller)

    for {
        conn, e := s.listener.Accept()
        if e != nil {
            if ne, ok := e.(net.Error); ok && ne.Temporary() {
                log.Errorf("accept temp err: %v", ne)
                continue
            }

            log.Errorf("accept err: %v", e)
            return
        }
        fd := SocketFD(conn)

        sc := NewTCPConn(fd, conn, s, epoller)
        s.AddConn(fd, sc)

        err := epoller.Add(fd)
        if err != nil {
            log.Errorf("failed to add connection %v", err)
            sc.Close()
        }
        log.Infof("New connection fd[%d] incoming <%v>", fd, conn.RemoteAddr())
    }

}

func (s *Server) Start(epoller *epoll) {
    defer func() {
        log.Debugf("exit start")
        s.Close()
    }()

    for {
        fds, err := epoller.Wait()
        if err != nil {
            log.Errorf("failed to epoll wait %v", err)
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
                log.Errorf("fd[%d] not find int server aaaaaaaa", fd)
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

    if s.config.EpollNum < 0 {
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
    conns := s.conns.GetAll()
    for _, c := range conns {
        if err := c.Write(msg); err != nil {
            log.Errorf("broadcast error %v\n", err)
        }
    }
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
    s.listener.Close()
    log.Debugln("Server Has Close")
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
