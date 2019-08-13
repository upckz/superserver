package test

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
    "net"
    "superserver/until/common"
    log "superserver/until/czlog"
    "superserver/until/timingwheel"
    "sync"
    "time"
)

const (
    HEAET_TIMER_OUT int = iota
)

type GetAddrFunc func() string

type Client struct {
    name      string
    fd        int
    id        uint64
    addr      string
    svrType   int32
    svid      int32
    gameid    int32
    level     int32
    instance  *Instance
    isConnect *common.AtomicBoolean
    conn      net.Conn
    once      *sync.Once

    cache []byte //recv cache 接受缓存

    hbTimeout  time.Duration
    heartTimer *timingwheel.Timer
    timerWheel *timingwheel.TimingWheel
}

func NewClient(i *Instance, cfg *ClientConfig, idex uint64) *Client {

    c := &Client{
        id:         idex,
        once:       &sync.Once{},
        isConnect:  common.NewAtomicBoolean(false),
        hbTimeout:  cfg.HbTimeout,
        gameid:     cfg.Gameid,
        level:      cfg.Level,
        svid:       cfg.Svid,
        svrType:    cfg.ServerType,
        instance:   i,
        cache:      make([]byte, 0),
        timerWheel: i.timerWheel,
    }
    c.SetAddr(fmt.Sprintf("%s:%d", cfg.Ip, cfg.Port))

    if c.Connect() {
        return c
    }

    return nil
}

func (c *Client) SetInstance(i *Instance) {
    c.instance = i
}

//获取客户端连接的服务器类型
func (c *Client) GetSvrType() int32 {
    return c.svrType
}

func (c *Client) GetSvid() int32 {
    return c.svid
}

func (c *Client) GetGameid() int32 {
    return c.gameid
}

func (c *Client) GetLevel() int32 {
    return c.level
}

//设置名称
func (c *Client) SetName(name string) {
    c.name = name
}

//获取名称
func (c *Client) GetName() string {
    return c.name
}

//获取客户端的id
func (c *Client) GetFD() int {
    return c.fd
}

//检查是否已经连接好了
func (c *Client) HasConnect() bool {
    return c.isConnect.Get()
}

//ip地址
func (c *Client) SetAddr(addr string) {
    c.addr = addr
}

func (c *Client) GetAddr() string {
    return c.addr
}

func (c *Client) GetConn() net.Conn {
    return c.conn
}

func (c *Client) Connect() bool {

    if c.HasConnect() {
        return true
    }
    conn, err := net.DialTimeout("tcp", c.addr, 25*time.Second)
    c.conn = conn
    if err != nil {
        log.Errorf(c.name+" Connect failed:%v, %v", c.addr, err)
        return false
    }
    c.fd = SocketFD(conn)

    c.isConnect.Set(true)
    c.SetName(fmt.Sprintf("client [fd:%d->%v]", c.id, c.addr))

    c.LoginHall()

    return true
}

func (c *Client) ReConnect() {
    c.once.Do(func() {
        if c.HasConnect() {
            return
        }

    })
}

func (cw *Client) DoRecv() error {

    var buf = make([]byte, 1024*64)
    n, err := cw.conn.Read(buf)
    if err != nil {
        log.Errorf("connection close, fd[%d] <%v> err[%v]", cw.id, cw.conn.RemoteAddr(), err)
        cw.Close()
        return err
    }
    if n == 0 {
        log.Infof("connection close, fd[%d] <%v> ", cw.id, cw.conn.RemoteAddr())
        cw.Close()
        return errors.New("connnect close")
    }
    cw.cache = append(cw.cache, buf[0:n]...)
    for len(cw.cache) > 0 {
        pkglen := cw.ParsePacket(cw.cache)
        if pkglen < 0 {
            log.Errorf("fd[%d] [%s] recv error pkg", cw.id, cw.conn.RemoteAddr())
            cw.Close()
            return err
        } else if pkglen == 0 {
            break
        } else {
            err := cw.OnPacketComplete(cw.cache[0:pkglen])

            if err != nil {
                log.Errorf("fd[%d] ip[%s] input fatal error, ", cw.id, cw.conn.RemoteAddr().String())
                cw.Close()
                return err
            }
        }
        cw.cache = cw.cache[pkglen:]
    }
    return nil

}

func (cw *Client) ParsePacket(data []byte) int32 {

    n := int32(len(data))
    //need continue to recv buf
    if n < 4 {
        return 0
    }
    bufReader := bytes.NewReader(data[0:4])
    var pkglen int32
    err := binary.Read(bufReader, binary.BigEndian, &pkglen)
    //error to pkg
    if err != nil || pkglen > 65535 {
        return -1

    }
    // is one pkg
    if pkglen < 0 {
        return -1
    }

    //need continue to recv buf
    if n < pkglen {
        return 0
    }
    //one complete pkg
    return pkglen

}

func (cw *Client) OnPacketComplete(data []byte) error {
    inputPacket := NewNETInputPacket(data)
    if inputPacket == nil {
        return errors.New("OnPacketComplete err")
    }
    err := inputPacket.DecryptBuffer()
    if err != nil {
        log.Errorf(err.Error())
        return err
    }
    cw.ProcessDoneMsg(inputPacket)
    return nil
}

func (cw *Client) ProcessDoneMsg(msg *NETInputPacket) int {

    cmd := msg.GetCmd()
    log.Debugf("fd[%d] ip[%s] cmd[0x%02x]  ", cw.id, cw.conn.RemoteAddr().String(), cmd)
    switch cmd {
    case 0x101:
        cw.LoginHallResponse(msg)
    case 0x0200:
        cw.startHeartbeatTimer()
    }

    cw.instance.OnWritePacket(msg)

    return 0

}

func (c *Client) Close() {

    if c.HasConnect() {
        log.Errorf("fd[%d] [%s] close! ", c.id, c.conn.RemoteAddr().String())
        c.conn.Close()
    }
    c.instance.OnCloseCli(c)
    c.stopHeartbeatTimer()

}

func (cw *Client) StopAllTimer() {
    cw.stopHeartbeatTimer()

}

func (cw *Client) startHeartbeatTimer() {
    cw.stopHeartbeatTimer()
    cw.heartTimer = cw.timerWheel.AfterFunc(cw.hbTimeout, cw.ProcessTimeOut, HEAET_TIMER_OUT)
}

func (cw *Client) stopHeartbeatTimer() {
    if cw.heartTimer != nil {
        cw.heartTimer.Stop()
    }
    cw.heartTimer = nil

}

func (cw *Client) ProcessTimeOut(timeType int) {

    log.Debugf("ProcessTimeOut|  cw.id[%d]", cw.id)
    switch timeType {
    case HEAET_TIMER_OUT:
        cw.SendHeartMsg()
    }
}

func (cw *Client) SendHeartMsg() {
    outPacket := NewNETOutputPacket()
    outPacket.Begin(0x0200)
    outPacket.End()
    data := outPacket.EncryptBuffer()
    cw.SendMessage(data)
    cw.startHeartbeatTimer()
}

func (cw *Client) LoginHall() {
    outPacket := NewNETOutputPacket()
    outPacket.Begin(0x0101)
    outPacket.WriteInt32(int32(cw.id))
    outPacket.WriteString("999999")
    outPacket.WriteString("gggggg")
    outPacket.WriteShort(int16(2))
    outPacket.End()
    data := outPacket.EncryptBuffer()

    cw.SendMessage(data)

}

func (cw *Client) LoginHallResponse(msg *NETInputPacket) {
    ret := msg.ReadByte()
    if ret == 0 {
        log.Debugf("fd[%d] ip[%s] loginHallSuccess ", cw.id, cw.conn.RemoteAddr().String())
    }
    cw.startHeartbeatTimer()
}

//写数据包
func (c *Client) SendMessage(pkg []byte) error {

    if c.conn != nil && c.HasConnect() {
        _, err := c.conn.Write(pkg)
        if err != nil {
            log.Errorf("fd[%d] ip[%s] send err [%v]", c.fd, c.conn.RemoteAddr().String(), err)
            return err
        }
        log.Debugf("fd[%d] ip[%s] send success ", c.id, c.conn.RemoteAddr().String())

    }

    return nil
}
