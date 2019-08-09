package socket

import (
    "bytes"
    "context"
    "encoding/binary"
    "fmt"
    "net"
    "strconv"
    "superserver/cmd"
    "superserver/until/common"
    "superserver/until/crypto/dh64"
    log "superserver/until/czlog"
    "sync"
    "time"

    "github.com/koangel/grapeTimer"
)

type GetAddrFunc func() string

type Client struct {
    name      string
    id        int
    addr      string
    svrType   int32
    svid      int32
    gameid    int32
    level     int32
    instance  *Instance
    isConnect *common.AtomicBoolean
    conn      net.Conn
    once      *sync.Once

    server *Server

    needReconnectFlag bool

    secertKey []byte //通讯密钥
    secert    bool   //通讯是否加密

    reconnectCount *common.AtomicUint64

    cache []byte //recv cache 接受缓存

    hbTimer   *time.Timer
    hbTimeout time.Duration

    connectTimer int

    sendCh chan *common.Message

    ctx    context.Context
    cancel context.CancelFunc
}

func NewClient(ctx context.Context, idx int) *Client {

    c := &Client{
        once:           &sync.Once{},
        reconnectCount: common.NewAtomicUint64(0),
        isConnect:      common.NewAtomicBoolean(false),
        id:             idx,
        sendCh:         make(chan *common.Message, 1000),
        conn:           nil,
    }
    c.ctx, c.cancel = context.WithCancel(ctx)
    return c
}

func (c *Client) SetInstance(i *Instance) {
    c.instance = i
}

//设置连接服务器type
func (c *Client) SetSvrType(svrType int32) {
    c.svrType = svrType
}

//获取客户端连接的服务器类型
func (c *Client) GetSvrType() int32 {
    return c.svrType
}

func (c *Client) SetSvid(svid int32) {
    c.svid = svid
}

func (c *Client) GetSvid() int32 {
    return c.svid
}

func (c *Client) SetGameid(gameid int32) {
    c.gameid = gameid
}

func (c *Client) GetGameid() int32 {
    return c.gameid
}

func (c *Client) SetLevel(level int32) {
    c.level = level
}

func (c *Client) GetLevel() int32 {
    return c.level
}

func (c *Client) SetNeedReconnectFlag(sflag bool) {
    c.needReconnectFlag = sflag
}

//设置心跳
func (c *Client) SetHbTimer(ti int32) {

    c.hbTimeout = time.Duration(ti) * time.Second
    c.hbTimer = time.NewTimer(c.hbTimeout)
    if ti == 0 {
        c.hbTimer.Stop()
    }
}

//设置秘钥
func (c *Client) SetSecert(s bool) {
    c.secert = s
}

//设置服务
func (c *Client) SetServer(server *Server) {
    c.server = server

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
func (c *Client) GetID() int {
    return c.id
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

func (c *Client) Connect() {

    c.once.Do(func() {

        if c.HasConnect() {
            return
        }
        conn, err := net.DialTimeout("tcp", c.addr, 25*time.Second)
        c.conn = conn
        if err != nil {
            log.Errorf(c.name+" Connect failed:%v, %v", c.addr, err)
            c.Close()
            return
        }
        if c.secert == true {
            err := c.BulidConntinon()
            if err != nil {
                log.Errorf(c.name+" Connect secert failed:%v, %v", c.addr, err)
                c.Close()
                return
            }

        }
        c.isConnect.Set(true)
        c.SetName(fmt.Sprintf("client [id:%d->%v]", c.id, c.addr))
        log.Infof("client[%d] connect ip[%s] success", c.id, c.conn.RemoteAddr().String())

        c.instance.AddMapCli(c)
        c.SendHeartbeatMsg()
        go c.RecvLoop()

    })
}

func (c *Client) BulidConntinon() error {
    if c.secert == true {
        wb := make([]byte, 7)
        binary.BigEndian.PutUint32(wb[:4], uint32(7))
        wb[4] = byte(1)
        wb[5] = byte('S')
        wb[6] = byte('W')
        _, err := c.conn.Write(wb)

        if err != nil {
            return err
        }
        buff := make([]byte, 1024)
        n, err := c.conn.Read(buff)
        if err != nil {
            return err
        }
        buff = buff[0:n]

        head := &common.PackageHead{
            Data: make([]byte, len(buff)-5),
        }

        bufReader := bytes.NewReader(buff)
        err = binary.Read(bufReader, binary.BigEndian, &head.Length)
        if err != nil {
            log.Errorf("client[%d] %s read headlen err[%v]", c.id, c.conn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Ack)
        if err != nil {
            log.Errorf("client[%d] %s read head ack err[%v]", c.id, c.conn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Data)
        if err != nil {
            log.Errorf("client[%d] %s read head ack err[%v]", c.id, c.conn.RemoteAddr().String(), err.Error())
            return err
        }
        if head.Ack == 2 {

            privateKey, publicKey := dh64.KeyPair()

            wb := make([]byte, 13)
            binary.BigEndian.PutUint32(wb[:4], uint32(13))
            wb[4] = byte(3)
            binary.BigEndian.PutUint64(wb[5:], uint64(publicKey))

            if _, err := c.conn.Write(wb); err != nil {
                return err
            }

            serverPublicKey := uint64(binary.BigEndian.Uint64(head.Data))

            secert, err := dh64.Secret(privateKey, uint64(serverPublicKey))
            if err != nil {
                log.Errorf("clientId[%d] %s, DH64 secert Error:%v", c.id, c.conn.RemoteAddr().String(), err.Error())
                return err
            }
            secertStr := strconv.FormatUint(secert, 10)
            secertMaxIndex := uint8(len(secertStr) - 1)
            var k uint8 = 0
            buffer := bytes.NewBufferString(secertStr)
            for i := secertMaxIndex + 1; i < 32; i++ {
                if k > secertMaxIndex {
                    k = 0
                }
                buffer.WriteByte(secertStr[k])
                k++
            }
            secertStr = buffer.String()

            var keyBuffer bytes.Buffer
            for i := 0; i < 32; i = i + 2 {
                tempInt, _ := strconv.Atoi(secertStr[i : i+2])
                keyBuffer.WriteRune(rune(tempInt))
            }

            log.Infof("clientId[%d] connectHandler DH64 clientPublicKey[%v] serverPublicKey[%v] secert[%v] secertStr[%v] local[%s] ip[%v] connect",
                c.id, publicKey, serverPublicKey, secert, secertStr, c.conn.LocalAddr(), c.conn.RemoteAddr().String())

            sw := keyBuffer.Bytes()
            c.secertKey = make([]byte, len(sw))
            copy(c.secertKey, sw)
            c.instance.NoticeToConnect(c)

        }

    }
    return nil
}

func (cw *Client) RecvLoop() {
    defer func() {
        if p := recover(); p != nil {
            log.Errorln(cw.name + " " + common.GlobalPanicLog(p))
        }

        log.Debugln(cw.name + "runWrite go-routine exited")
        cw.Close()
    }()

    if cw.conn == nil {
        return
    }
    rclose := make(chan int, 1)
    go func() {

        defer func() {
            rclose <- 1
        }()

        var buf = make([]byte, 1024*64)
        for {
            n, err := cw.conn.Read(buf)
            if n == 0 || err != nil {
                log.Infof("client[%d] <%v>  err[%v]", cw.id, cw.conn.RemoteAddr(), err)
                cw.Close()
                return
            } else {
                cw.cache = append(cw.cache, buf[0:n]...)
                for len(cw.cache) > 0 {
                    pkglen := common.ParsePacket(cw.cache)
                    if pkglen == -1 {
                        log.Errorf("client[%d] <%v> recv error pkg", cw.id, cw.conn.RemoteAddr())
                        cw.Close()
                        return
                    } else if pkglen == 0 {
                        break
                    } else {
                        msg, err := common.OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey)
                        if err != nil || msg == nil {
                            log.Errorf("client[%d] ip[%s] input fatal error", cw.id, cw.conn.RemoteAddr().String())
                            cw.Close()
                            return
                        }

                        cw.ProcessDoneMsg(msg)
                        cw.cache = cw.cache[pkglen:]
                    }

                }
            }
        }
    }()

    for {
        select {
        case <-cw.ctx.Done():
            log.Debugf(cw.name + " close runWrite...\n")
            return
        case <-cw.hbTimer.C:
            cw.SendHeartbeatMsg()
            if cw.hbTimeout > 0 {
                cw.hbTimer.Reset(cw.hbTimeout)
            }

        case pkt, ok := <-cw.sendCh:
            if ok && pkt != nil {
                err := cw.SendMessage(pkt)
                if err != nil {
                    log.Errorf(cw.name+" error writing data %v\n", err)
                    return
                }
            }

        case <-rclose:
            return

        }
    }
}

func (cw *Client) ProcessDoneMsg(msg *common.Message) int {

    wrapper := &CliMsgWrapper{
        ID:  cw.GetID(),
        Msg: msg,
    }
    cw.instance.OnWritePacket(wrapper)
    return 0

}

func (c *Client) Close() {
    c.cancel()
    c.instance.OnCloseCli(c)
    if c.HasConnect() {
        log.Errorf("client[%d] [%s] close! ", c.id, c.conn.RemoteAddr().String())
        c.conn.Close()
    }
    c.isConnect.Set(false)
    c.stopHeartbeatTimer()
    if c.needReconnectFlag {
        c.startReConnectTimer()
    }
}

func (cw *Client) StopAllTimer() {
    cw.stopHeartbeatTimer()
    cw.startReConnectTimer()
}

func (cw *Client) stopHeartbeatTimer() {
    cw.hbTimeout = 0
}

func (cw *Client) startReConnectTimer() {
    cw.stopReConnectTimer()
    cw.reconnectCount.AddAndGet(1)
    cw.connectTimer = grapeTimer.NewTickerOnce(int(cw.reconnectCount.Get())*100, cw.ProcessTimeOut, RECONNECT_TIMER_OUT)
}
func (cw *Client) stopReConnectTimer() {
    grapeTimer.StopTimer(cw.connectTimer)
}

func (cw *Client) ProcessTimeOut(timeType int) {

    log.Debugf("ProcessTimeOut|  cw.id[%d]", cw.id)
    switch timeType {
    case HEAET_TIMER_OUT:
        cw.SendHeartbeatMsg()
    case RECONNECT_TIMER_OUT:
        if cw.needReconnectFlag && !cw.HasConnect() {
            go func() {
                time.Sleep(1 * time.Second)
                cw.once = &sync.Once{}
                go cw.Connect()
            }()
        }
    }
}

func (c *Client) SendHeartbeatMsg() {
    if !c.HasConnect() {
        return
    }
    msg := common.NewMessage(cmd.MsgHeartbeat, int32(c.id), make([]byte, 0))
    log.Debugf("client[%d] ip[%s] send heart [%v]", c.id, c.conn.RemoteAddr().String(), msg)
    c.WritePacket(msg)

}

//写数据包
func (c *Client) WritePacket(pkt *common.Message) {
    if pkt != nil && c.sendCh != nil {
        sendIsFull := len(c.sendCh) == cap(c.sendCh)
        hasConnect := c.HasConnect()
        if !sendIsFull || hasConnect {
            if sendIsFull && hasConnect {
                log.Warnln("Write Seq Has Full")
            }
            c.sendCh <- pkt
        }
    }
}

func (c *Client) SendMessage(msg *common.Message) error {

    var pkg []byte
    var err error
    if c.secert == true {

        pkg, err = common.EnBinaryPackage(msg, c.secertKey)
        if err != nil {
            return err
        }
    } else {
        pkg, err = common.Encode(msg)
        if err != nil {
            return err
        }
    }
    if c.conn != nil && c.HasConnect() {
        _, err := c.conn.Write(pkg)
        if err != nil {
            return err
        }
    }

    return nil
}
