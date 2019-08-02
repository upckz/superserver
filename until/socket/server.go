package socket

import (
    "context"
    "strconv"
    "net"
    "fmt"
    "time"

    log "superserver/until/czlog"
    "superserver/until/common"

    "syscall"
    "github.com/libp2p/go-reuseport"
    "github.com/koangel/grapeTimer"
)

const (
    SocketTypeServer int32 = iota
    SocketTypeClient
)

//Config use for custom the Server
type Config struct {
    IP                 string
    Port               int32
    MsgCh              chan *MessageWrapper
    OnConnect          OnConnectFunc
    OnClose            OnCloseFunc
    EpollNum           int32
    HbTimeout          int32
    Secert             bool
    Ctx                context.Context
}

//Server serve a tcp service
type Server struct {
    conns         *ConnMap
    config        *Config
    name          string
    ctx           context.Context
    cancel        context.CancelFunc
    secert        bool


}

//NewServer create a Server instance with conf
func NewServer(conf *Config) *Server {
    s := &Server{
        config:        conf,
        conns:         NewConnMap(),
        secert:        conf.Secert,
    }
    s.ctx, s.cancel = context.WithCancel(conf.Ctx)
    s.SetName(fmt.Sprint(s))
    return s
}



func (s *Server)startEpoll(index int){

   defer func() {
        if p := recover(); p != nil {
            log.Errorln(common.GlobalPanicLog(p))
        }
        log.Debugf("exit[%d] startEpoll", index)
    }()

    addr := s.config.IP + ":" + strconv.Itoa(int(s.config.Port))

    ln, err := reuseport.Listen("tcp", addr)
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

    log.Infof("Sever[%s] start listen on: %v", s.name, ln.Addr())
   
    go s.Start(epoller,index)

    for {
        conn, e := ln.Accept()
        if e != nil {
            if ne, ok := e.(net.Error); ok && ne.Temporary() {
                log.Errorf("accept temp err: %v", ne)
                continue
            }

            log.Errorf("accept err: %v", e)
            return
        }
        fd, err := epoller.Add(conn)
        if err != nil {
            log.Errorf("failed to add connection %v", err)
            conn.Close()
        }
        sc := NewTCPConn(fd, conn,  s)
        if _, ok := s.conns.Get(fd) ; ok{
            log.Errorf("fd[%d] is used, error", fd)
        }
        s.conns.Put(fd, sc)
        log.Infof("New connection connId[%d] incoming <%v>", fd, conn.RemoteAddr())
    }

}


func (s *Server)Start(epoller *epoll, index int) {
    defer func() {
        log.Debugf("exit[%d] start", index)
        s.Close()
    }()

   for {
        connections, fdList , err := epoller.Wait()
        if err != nil {
            log.Errorf("failed to epoll wait %v", err)
            continue
        }
      
        nums := len(connections)
        for i := 0; i <  nums; i++ {
            conn := connections[i]
            if conn == nil {
                break
            }
            fd := fdList[i]

            sc, ok := s.GetConn(fd)
            if ok {
                sc.Recv(epoller, conn)
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
func (s *Server) Run()  {
    setLimit()

    //开启定时器500 ms 单位轮询
    grapeTimer.InitGrapeScheduler(20*time.Microsecond, true)
    grapeTimer.CDebugMode = false  //设置启用日志调试模式，建议正式版本关闭他
    grapeTimer.UseAsyncExec = true  //开启异步调度模式，在此模式下 timer执行时会建立一个go，不会阻塞其他timer执行调度，建议开启
    if s.config.EpollNum < 0 {
        s.config.EpollNum = 1
    }
    for i := 0; i < int(s.config.EpollNum); i++ {
        go s.startEpoll(i)
    }

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
    log.Debugln("Server Has Close")
}

func (s *Server)CloseConn(id int) {
    conn , has := s.GetConn(id)
    if has && conn != nil {
        conn.Close()
    }
}

func(s *Server)GetTotalConnect() int{
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
