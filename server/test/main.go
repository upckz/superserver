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
    "github.com/sirupsen/logrus"
    "superserver/until/common"
    log "superserver/until/log"
    "superserver/until/socket"
)

//cpu 自动确认使用核数
func init() {
    runtime.GOMAXPROCS(runtime.NumCPU())
}

type Config struct {
    SvrID     int32
    SverType  int32
    IP        string
    Port      int32
    Gameid    int32
    HbTimeout time.Duration
}

type Server struct {
    svid                int32
    sverType            int32
    cfg                 *Config
    sigCh               chan os.Signal
    internalSvrInstance *socket.Server
    cliMgr              *socket.Instance
    readCh              chan *socket.MessageWrapper

    ctx    context.Context
    cancel context.CancelFunc
}

func NewServer(cfg *Config) *Server {

    s := &Server{
        svid:     cfg.SvrID,
        sverType: cfg.SverType,
        cfg:      cfg,
        readCh:   make(chan *socket.MessageWrapper, 10000),
        sigCh:    make(chan os.Signal, 1),
    }
    s.ctx, s.cancel = context.WithCancel(context.Background())

    svrCfg := &socket.Config{
        IP:        cfg.IP,
        Port:      cfg.Port,
        MsgCh:     s.readCh,
        HbTimeout: 30 * time.Second,
        Secert:    true,
        Ctx:       s.ctx,
        EpollNum:  2,
        OnConnect: func(netid int) {
            log.WithFields(log.Fields{
                "id": netid,
            }).Info("Internal Server connected")
        },
        OnClose: func(netid int) {
            log.WithFields(log.Fields{
                "id": netid,
            }).Info("Internal Server Closed")
        },
    }
    s.cliMgr = socket.NewClientInstance(s.ctx, 10000)
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
    log.WithFields(log.Fields{}).Info("server Stop")

}

func (s *Server) dispatchClientMsg() {
    defer func() {
        if p := recover(); p != nil {
            log.WithFields(log.Fields{}).Panic(common.GlobalPanicLog(p))
        }
        log.WithFields(log.Fields{
            "svid": s.cfg.SvrID,
        }).Info("dispatchClientMsg close success")

    }()
    if s.readCh == nil {
        return
    }
    for {

        select {
        case <-s.ctx.Done():
            return
        case rawMsg, ok := <-s.readCh:
            if ok {
                log.WithFields(log.Fields{}).Info(rawMsg)
            }
        case pkt, ok := <-s.cliMgr.OnReadPacket():
            if ok {
                log.WithFields(log.Fields{}).Info(pkt)
            }
        case id, ok := <-s.cliMgr.OnConnect():
            if ok {
                log.WithFields(log.Fields{
                    "clinetid": id,
                }).Info("connected success")
            }
        }
    }
}

func main() {
    defer func() {
        if err := recover(); err != nil {
            log.WithFields(log.Fields{}).Panic(common.GlobalPanicLog(err))
        }
    }()

    logPath := flag.String("log", "./log/", "log file path")
    logLevel := flag.String("logLevel", "DEBUG", "日志等级:INFO,PANIC,FATAL,ERROR,WARN,DEBUG")
    strIp := flag.String("ip", "0.0.0.0", "listen ip")
    strPort := flag.Int("port", 9100, "listen port")
    svid := flag.Int("svid", 1, "server svid. start from 1")
    flag.Parse()

    initLog(*logPath, *logLevel)

    if *strIp == "" {
        fmt.Println("You Need Specify The ip with -ip")
        return
    }
    if *svid == 0 {
        fmt.Println("you must specify the server svid with -svid")
        return
    }
    if *strPort == 0 {
        fmt.Println("you must specify the server port with -port")
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
    log.WithFields(log.Fields{}).Info("server exit")
}

func initLog(path string, logLevel string) {
    switch logLevel {
    case "INFO":
        log.SetLevel(logrus.InfoLevel)
    case "PANIC":
        log.SetLevel(logrus.PanicLevel)
    case "FATAL":
        log.SetLevel(logrus.FatalLevel)
    case "ERROR":
        log.SetLevel(logrus.ErrorLevel)
    case "WARN":
        log.SetLevel(logrus.WarnLevel)
    default:
        log.SetLevel(logrus.DebugLevel)
    }
    log.SetPath(path, "main", 5)

}
