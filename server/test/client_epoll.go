package main

import (
    "context"
    "flag"
    "fmt"
    "github.com/koangel/grapeTimer"
    "net/http"
    _ "net/http/pprof"
    "os"
    "runtime"
    "superserver/example/epoller_client"
    "superserver/until/common"
    log "superserver/until/czlog"
    "superserver/until/epoll"
    "time"
)

//cpu 自动确认使用核数
func init() {
    runtime.GOMAXPROCS(runtime.NumCPU())

}

func runlog(path string) {
    path = fmt.Sprintf("%s/testLog", path)
    defer log.Start(log.LogFilePath(path), log.EveryDay, log.AlsoStdout, log.DebugLevel).Stop()

}

type Config struct {
    SvrID     int32
    SverType  int32
    IP        string
    Port      int32
    Gameid    int32
    HbTimeout int
}

type Server struct {
    sigCh  chan os.Signal
    cliMgr *epoller_client.Instance
    cfg    *Config
    sendCh chan *common.ResponseWrapper
    readCh chan *common.MessageWrapper
    ctx    context.Context
    cancel context.CancelFunc
}

func NewServer(cfg *Config) *Server {

    s := &Server{
        cfg:    cfg,
        sendCh: make(chan *common.ResponseWrapper, 100),
        readCh: make(chan *common.MessageWrapper, 100),
        sigCh:  make(chan os.Signal, 1),
    }

    s.ctx, s.cancel = context.WithCancel(context.Background())

    s.cliMgr = epoller_client.NewClientInstance(s.ctx, 3000)

    return s
}

func (s *Server) mkClient() {
    epoller, err := epoll.MkEpoll()
    if err != nil {
        panic(err)
    }

    cfg := &epoller_client.ConfigOfClient{
        HbTimeout:     20,
        Ip:            s.cfg.IP,
        Port:          s.cfg.Port,
        Secert:        false,
        ReconnectFlag: false,
    }

    go s.Start(epoller)

    for i := 0; i < 20000; i++ {
        s.cliMgr.AddClientWith(cfg, epoller, s.cfg.SvrID*200000) //server作为客户端 ，管理所有的客户端连接请求, 是否加密访问

    }

}

func (s *Server) Start(epoller *epoll.Epoll) {
    defer func() {
        log.Debugf("exit start")
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
            client := s.cliMgr.GetMapCli(fd)
            if client != nil {
                client.Recv()
                // if err == nil {
                //     epoller.Mode(fd)
                // }
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

func (s *Server) Run() {

    go s.runRead()

    s.mkClient()
}

func (s *Server) runRead() {

    defer func() {
        if p := recover(); p != nil {
            log.Errorln(common.GlobalPanicLog(p))
        }
        log.Debugf("runRead close success")
    }()
    log.Debugf("begin runRead....")

    for {
        select {
        case <-s.ctx.Done():
            log.Debugf("receiving cancel signal from conn")
            return
        case pkt, ok := <-s.cliMgr.OnReadPacket():
            if ok {
                log.Debugf("OnReadPacket....pkt[%v]", pkt)
            }

        case id, ok := <-s.cliMgr.OnConnect():
            if ok {
                log.Debugf("OnConnect[%d]", id)
            }
        }
    }
}

func main() {
    defer func() {
        if err := recover(); err != nil {
            log.Errorln(common.GlobalPanicLog(err))
        }
    }()

    go func() {
        if err := http.ListenAndServe(":6061", nil); err != nil {
            log.Fatalf("pprof failed: %v", err)
        }
    }()

    logPath := flag.String("log", "./log/", "path for log file directory")
    strIp := flag.String("ip", "172.16.10.51", "listen ip")
    //strIp := flag.String("ip", "192.168.56.101", "listen ip")
    strPort := flag.Int("port", 9100, "listen port")
    svid := flag.Int("svid", 1, "server svid. start from 1")

    flag.Parse()

    runlog(*logPath)

    if *strIp == "" {
        log.Fatalln("You Need Specify The ip with -ip")
        return
    }
    if *svid == 0 {
        log.Fatalln("you must specify the server svid with -svid")
        return
    }
    if *strPort == 0 {
        log.Fatalln("you must specify the server port with -port")
        return
    }
    log.Debugf("sss=%d", *svid)
    //开启定时器500 ms 单位轮询
    grapeTimer.InitGrapeScheduler(100*time.Microsecond, true)
    grapeTimer.CDebugMode = false  //设置启用日志调试模式，建议正式版本关闭他
    grapeTimer.UseAsyncExec = true //开启异步调度模式，在此模式下 timer执行时会建立一个go，不会阻塞其他timer执行调度，建议开启

    cfg := &Config{
        SvrID:    int32(*svid),
        SverType: 1,
        IP:       string(*strIp),
        Port:     int32(*strPort),
    }

    server := NewServer(cfg)
    go server.Run()
    common.WaitSignalSynchronized()
    log.Errorln("server Exit!")

}
