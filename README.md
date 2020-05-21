    # superserver
    可处理百万级并发
    可同时做服务器 和 client

    多epoll处理， 可设置监听epoll
    1.简单创建：
 
    //配置信息  
    cfg := &Config{ 
        SvrID:    int32(*svid), //服务器id，
        SverType: 1,               //服务器类型
        IP:       string(*strIp),   //监听ip和port
        Port:     int32(*strPort),
    }

    //调用即可
    server := NewServer(cfg)
    server.Run()
    server.Stop()
    
    2. Serve 开始进行监听，可设置多个协城监听epoll事件。
    func (s *Server) Run() {
        for i := 0; i < int(s.config.EpollNum); i++ {
            go s.startEpoll()
        }

    }


    具体实现： 看test/main.go
    test 为测试目录

    依赖库：
    github.com/koangel/grapeTimer
    github.com/sirupsen/logrus
    github.com/libp2p/go-reuseport
    golang.org/x/sys/unix
