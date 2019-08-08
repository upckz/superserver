package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "runtime"
    "time"
    //_ "net/http/pprof"
    //"net/http"
    "github.com/koangel/grapeTimer"
    "superserver/until/common"
    log "superserver/until/czlog"
    "superserver/until/socket"
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
    HbTimeout int32
}

type Server struct {
    svid                int32
    sverType            int32
    cfg                 *Config
    sigCh               chan os.Signal
    internalSvrInstance *socket.Server

    sendCh chan *socket.ResponseWrapper
    readCh chan *socket.MessageWrapper

    ctx    context.Context
    cancel context.CancelFunc
}

func NewServer(cfg *Config) *Server {

    s := &Server{
        svid:     cfg.SvrID,
        sverType: cfg.SverType,
        cfg:      cfg,
        sendCh:   make(chan *socket.ResponseWrapper, 1000),
        readCh:   make(chan *socket.MessageWrapper, 1000),
        sigCh:    make(chan os.Signal, 1),
    }

    s.ctx, s.cancel = context.WithCancel(context.Background())

    svrCfg := &socket.Config{
        IP:        cfg.IP,
        Port:      cfg.Port,
        MsgCh:     s.readCh,
        HbTimeout: 15,
        Secert:    false,
        Ctx:       s.ctx,
        EpollNum:  10,
        OnConnect: func(netid int) {
            log.Debugf("Internal Server Connection[id:%d] connected", netid)
        },
        OnClose: func(netid int) {
            log.Debugf("Internal Server Connection[id:%d] Closed", netid)
        },
    }
    s.internalSvrInstance = socket.NewServer(svrCfg)
    return s
}

func (s *Server) Run() {

    //开启定时器500 ms 单位轮询
    grapeTimer.InitGrapeScheduler(100*time.Microsecond, true)
    grapeTimer.CDebugMode = false  //设置启用日志调试模式，建议正式版本关闭他
    grapeTimer.UseAsyncExec = true //开启异步调度模式，在此模式下 timer执行时会建立一个go，不会阻塞其他timer执行调度，建议开启

    go s.internalSvrInstance.Run()
    go s.dispatchClientMsg()
}

//Stop 关闭server
func (s *Server) Stop() {
    s.cancel()
    log.Errorf("(S_%d)Stop", s.cfg.SvrID)

}

func (s *Server) dispatchClientMsg() {
    defer func() {
        if p := recover(); p != nil {
            log.Errorln(common.GlobalPanicLog(p))
        }
        log.Debugf("dispatchClientMsg close success")
    }()
    if s.readCh == nil {
        return
    }
    log.Debugf("begin dispatchClientMsg....")
    for {

        select {
        case <-s.ctx.Done():
            return
        case rawMsg, ok := <-s.readCh:
            if ok {
                log.Debugf("msg[%v]", rawMsg)
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

    logPath := flag.String("log", "./log/", "path for log file directory")
    strIp := flag.String("ip", "0.0.0.0", "listen ip")
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

    cfg := &Config{
        SvrID:    int32(*svid),
        SverType: 1,
        IP:       string(*strIp),
        Port:     int32(*strPort),
    }

    server := NewServer(cfg)
    server.Run()
    common.WaitSignalSynchronized()
    server.Stop()
    time.Sleep(1 * time.Second)
    log.Errorln("server Exit!")
}
