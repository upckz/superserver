package main

import (
    "context"
    "flag"
    "fmt"
    // "net/http"
    // _ "net/http/pprof"
    "github.com/sirupsen/logrus"
    "os"
    "runtime"
    "superserver/until/common"
    "superserver/until/log"
    "superserver/until/socket"
    "time"
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
    sigCh  chan os.Signal
    cliMgr *socket.Instance
    cfg    *Config
    ctx    context.Context
    cancel context.CancelFunc
}

func NewServer(cfg *Config) *Server {

    s := &Server{
        cfg:   cfg,
        sigCh: make(chan os.Signal, 1),
    }

    s.ctx, s.cancel = context.WithCancel(context.Background())

    s.cliMgr = socket.NewClientInstance(s.ctx, 3000)

    return s
}

func (s *Server) Run() {

    cfg := &socket.ClientConfig{
        HbTimeout:     20 * time.Second,
        Ip:            s.cfg.IP,
        Port:          s.cfg.Port,
        Secert:        true,
        ReconnectFlag: false,
    }
    go s.runRead()

    for i := 0; i < 5000; i++ {
        s.cliMgr.AddClientWith(cfg, int(s.cfg.SvrID*20000)) //server作为客户端 ，管理所有的客户端连接请求, 是否加密访问

    }
}

func (s *Server) runRead() {

    defer func() {
        if p := recover(); p != nil {
            log.WithFields(log.Fields{}).Panic(p)
        }
        log.WithFields(log.Fields{}).Info("runRead close success")
    }()

    for {
        select {
        case <-s.ctx.Done():
            log.WithFields(log.Fields{}).Info("receiving cancel signal from conn")
            //log.Debugf("receiving cancel signal from conn")
            return
        case pkt, ok := <-s.cliMgr.OnReadPacket():
            if ok {
                log.WithFields(log.Fields{}).Info(pkt)
                // log.Debugf("OnReadPacket....pkt[%v]", pkt)
            }

        case id, ok := <-s.cliMgr.OnConnect():
            if ok {
                log.WithFields(log.Fields{
                    "clinetid": id,
                }).Info("connected")
            }
        }
    }
}

func main() {
    defer func() {
        if err := recover(); err != nil {
            log.WithFields(log.Fields{}).Panic(err)
        }
    }()

    // go func() {
    //     if err := http.ListenAndServe(":6061", nil); err != nil {
    //         log.Fatalf("pprof failed: %v", err)
    //     }
    // }()

    logPath := flag.String("log", "./log/", "log file path")
    logLevel := flag.String("logLevel", "DEBUG", "日志等级:INFO,PANIC,FATAL,ERROR,WARN,DEBUG")
    strIp := flag.String("ip", "172.16.10.51", "listen ip")
    //strIp := flag.String("ip", "192.168.56.102", "listen ip")
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
    go server.Run()
    common.WaitSignalSynchronized()
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
    log.SetPath(path, "client", 5)

}
