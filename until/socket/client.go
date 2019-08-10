package socket

import (
    "bytes"
    "encoding/binary"
    "errors"
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

    needReconnectFlag bool

    secertKey []byte //通讯密钥
    secert    bool   //通讯是否加密

    reconnectCount *common.AtomicUint64

    cache []byte //recv cache 接受缓存

    hbTimeout int32
    hbTimer   int

    connectTimer int
    heartTimer   int
    sendCh       chan *Message

    hasRecvFlag bool //recv flag('SW') to change to publicKey
}

func NewClient(i *Instance, cfg *ClientConfig) *Client {

    c := &Client{
        once:              &sync.Once{},
        reconnectCount:    common.NewAtomicUint64(0),
        isConnect:         common.NewAtomicBoolean(false),
        secert:            cfg.Secert,
        hbTimeout:         cfg.HbTimeout,
        secertKey:         make([]byte, 0),
        gameid:            cfg.Gameid,
        level:             cfg.Level,
        svid:              cfg.Svid,
        svrType:           cfg.ServerType,
        instance:          i,
        cache:             make([]byte, 0),
        hasRecvFlag:       false,
        needReconnectFlag: cfg.ReconnectFlag,
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

func (c *Client) SetNeedReconnectFlag(sflag bool) {
    c.needReconnectFlag = sflag
}

//设置心跳
func (c *Client) SetHbTimer(ti int32) {

    c.hbTimeout = ti

}

//设置秘钥
func (c *Client) SetSecert(s bool) {
    c.secert = s
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
    if c.secert == true {
        err := c.BulidConntinon()
        if err != nil {
            log.Errorf(c.name+" Connect secert failed:%v, %v", c.addr, err)
            conn.Close()
            return false
        }
    }
    c.fd = SocketFD(conn)

    c.isConnect.Set(true)
    c.SetName(fmt.Sprintf("client [fd:%d->%v]", c.fd, c.addr))
    log.Infof("client fd[%d] connect ip[%s] success", c.fd, c.conn.RemoteAddr().String())
    c.SendHeartbeatMsg()
    return true
}

func (c *Client) ReConnect() {
    c.once.Do(func() {
        if c.HasConnect() {
            return
        }
        if !c.Connect() {
            c.startReConnectTimer()
        }
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

        head := &PackageHead{
            Data: make([]byte, len(buff)-5),
        }

        bufReader := bytes.NewReader(buff)
        err = binary.Read(bufReader, binary.BigEndian, &head.Length)
        if err != nil {
            log.Errorf("fd[%d] %s read headlen err[%v]", c.fd, c.conn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Ack)
        if err != nil {
            log.Errorf("fd[%d] %s read head ack err[%v]", c.fd, c.conn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Data)
        if err != nil {
            log.Errorf("fd[%d] %s read head ack err[%v]", c.fd, c.conn.RemoteAddr().String(), err.Error())
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
                log.Errorf("fd[%d] %s, DH64 secert Error:%v", c.fd, c.conn.RemoteAddr().String(), err.Error())
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

            log.Infof("fd[%d] connectHandler DH64 clientPublicKey[%v] serverPublicKey[%v] secert[%v] secertStr[%v] local[%s] ip[%v] connect",
                c.fd, publicKey, serverPublicKey, secert, secertStr, c.conn.LocalAddr(), c.conn.RemoteAddr().String())

            sw := keyBuffer.Bytes()
            c.secertKey = make([]byte, len(sw))
            copy(c.secertKey, sw)
            c.instance.NoticeToConnect(c)

        }

    }
    return nil
}

func (cw *Client) DoRecv() error {

    var buf = make([]byte, 1024*64)
    n, err := cw.conn.Read(buf)
    if err != nil {
        log.Errorf("connection close, fd[%d] <%v> err[%v]", cw.fd, cw.conn.RemoteAddr(), err)
        cw.Close()
        return err
    }
    if n == 0 {
        log.Infof("connection close, fd[%d] <%v> ", cw.fd, cw.conn.RemoteAddr())
        cw.Close()
        return errors.New("connnect close")
    }
    cw.cache = append(cw.cache, buf[0:n]...)
    for len(cw.cache) > 0 {
        pkglen := ParsePacket(cw.cache)
        if pkglen < 0 {
            log.Errorf("fd[%d] [%s] recv error pkg", cw.fd, cw.conn.RemoteAddr())
            cw.Close()
            return err
        } else if pkglen == 0 {
            break
        } else {
            msg, err := OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey)

            if err != nil || msg == nil {
                log.Errorf("fd[%d] ip[%s] input fatal error, ", cw.fd, cw.conn.RemoteAddr().String())
                cw.Close()
                return err
            } else {
                log.Debugf("fd[%d] msg[%v]", cw.fd, msg)
                cw.ProcessDoneMsg(msg)
            }

            // if cw.hasRecvFlag {
            //     msg, err := common.OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey)

            //     if err != nil || msg == nil {
            //         log.Errorf("fd[%d] ip[%s] input fatal error, ", cw.fd, cw.conn.RemoteAddr().String())
            //         cw.Close()
            //         return err
            //     } else {
            //         cw.ProcessDoneMsg(msg)
            //     }
            // } else {
            //     err := cw.BulidEncryptionConnction(cw.cache[0:pkglen])
            //     if err != nil {
            //         cw.Close()
            //         return err
            //     }
            // }
        }
        cw.cache = cw.cache[pkglen:]
    }
    return nil

}

// func (c *Client)SendAckMsg() {
//     if c.secert == true {
//         wb := make([]byte, 7)
//         binary.BigEndian.PutUint32(wb[:4], uint32(7))
//         wb[4] = byte(1)
//         wb[5] = byte('S')
//         wb[6] = byte('W')
//         _, err := c.conn.Write(wb)

//         if err != nil {
//             return err
//         }
//     }
// }

// func (cw *Client) BulidEncryptionConnction(buff []byte) error {
//     if !cw.hasRecvFlag && cw.secert {

//         head := &PackageHead{
//             data: make([]byte, len(buff)-5),
//         }

//         bufReader := bytes.NewReader(buff)
//         err := binary.Read(bufReader, binary.BigEndian, &head.length)
//         if err != nil {
//             log.Errorf("fd[%d] %s read headlen err[%v]", cw.fd, cw.conn.RemoteAddr().String(), err.Error())
//             return err
//         }
//         err = binary.Read(bufReader, binary.BigEndian, &head.ack)
//         if err != nil {
//             log.Errorf("fd[%d] %s read head ack err[%v]", cw.fd, cw.conn.RemoteAddr().String(), err.Error())
//             return err
//         }
//         err = binary.Read(bufReader, binary.BigEndian, &head.data)
//         if err != nil {
//             log.Errorf("fd[%d] %s read head ack err[%v]", cw.fd, cw.conn.RemoteAddr().String(), err.Error())
//             return err
//         }
//         if head.ack == 2 {

//             privateKey, publicKey := dh64.KeyPair()
//             wb := make([]byte, 13)
//             binary.BigEndian.PutUint32(wb[:4], uint32(13))
//             wb[4] = byte(3)
//             binary.BigEndian.PutUint64(wb[5:], uint64(publicKey))

//             if _, err := c.conn.Write(wb); err != nil {
//                 return err
//             }

//             serverPublicKey := uint64(binary.BigEndian.Uint64(head.data))

//             secert, err := dh64.Secret(privateKey, uint64(serverPublicKey))
//             if err != nil {
//                 log.Errorf("fd[%d] %s, DH64 secert Error:%v", c.id, c.conn.RemoteAddr().String(), err.Error())
//                 return err
//             }
//             secertStr := strconv.FormatUint(secert, 10)
//             secertMaxIndex := uint8(len(secertStr) - 1)
//             var k uint8 = 0
//             buffer := bytes.NewBufferString(secertStr)
//             for i := secertMaxIndex + 1; i < 32; i++ {
//                 if k > secertMaxIndex {
//                     k = 0
//                 }
//                 buffer.WriteByte(secertStr[k])
//                 k++
//             }
//             secertStr = buffer.String()

//             var keyBuffer bytes.Buffer
//             for i := 0; i < 32; i = i + 2 {
//                 tempInt, _ := strconv.Atoi(secertStr[i : i+2])
//                 keyBuffer.WriteRune(rune(tempInt))
//             }

//             log.Infof("fd[%d] connectHandler DH64 clientPublicKey[%v] serverPublicKey[%v] secert[%v] secertStr[%v] local[%s] ip[%v] connect",
//                 c.id, publicKey, serverPublicKey, secert, secertStr, c.conn.LocalAddr(), c.conn.RemoteAddr().String())

//             sw := keyBuffer.Bytes()
//             c.secertKey = make([]byte, len(sw))
//             copy(c.secertKey, sw)
//             c.instance.NoticeToConnect(c)
//             cw.hasRecvFlag = true

//         } else {
//             log.Errorf("fd[%d] %s, ack", cw.fd, cw.conn.RemoteAddr().String())
//             cw.Close()
//             return err
//         }
//     }

//     return nil
// }

func (cw *Client) ProcessDoneMsg(msg *Message) int {

    if msg.GetCmd() == cmd.MsgHeartbeat {
        cw.startHeartbeatTimer()
    }
    wrapper := &CliMsgWrapper{
        ID:  cw.GetFD(),
        Msg: msg,
    }
    cw.instance.OnWritePacket(wrapper)
    return 0

}

func (c *Client) Close() {

    if c.HasConnect() {
        log.Errorf("fd[%d] [%s] close! ", c.fd, c.conn.RemoteAddr().String())
        c.conn.Close()
    }
    c.instance.OnCloseCli(c)
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

func (cw *Client) startHeartbeatTimer() {
    cw.stopHeartbeatTimer()
    cw.heartTimer = grapeTimer.NewTickerOnce(int(cw.hbTimeout*1000), cw.ProcessTimeOut, HEAET_TIMER_OUT)
}

func (cw *Client) stopHeartbeatTimer() {
    grapeTimer.StopTimer(cw.heartTimer)
}

func (cw *Client) startReConnectTimer() {
    cw.stopReConnectTimer()
    cw.reconnectCount.AddAndGet(1)
    cw.connectTimer = grapeTimer.NewTickerOnce(int(cw.reconnectCount.Get())*500, cw.ProcessTimeOut, RECONNECT_TIMER_OUT)
}
func (cw *Client) stopReConnectTimer() {
    grapeTimer.StopTimer(cw.connectTimer)
}

func (cw *Client) ProcessTimeOut(timeType int) {

    log.Debugf("ProcessTimeOut|  cw.fd[%d]", cw.fd)
    switch timeType {
    case HEAET_TIMER_OUT:
        cw.SendHeartbeatMsg()
    case RECONNECT_TIMER_OUT:
        if cw.needReconnectFlag && !cw.HasConnect() {
            go func() {
                time.Sleep(1 * time.Second)
                cw.once = &sync.Once{}
                go cw.ReConnect()
            }()
        }
    }
}

func (c *Client) SendHeartbeatMsg() {
    if !c.HasConnect() {
        return
    }
    msg := NewMessage(cmd.MsgHeartbeat, int32(c.id), make([]byte, 0))
    log.Debugf("fd[%d] ip[%s] send heart [%v]", c.fd, c.conn.RemoteAddr().String(), msg)
    c.SendMessage(msg)
    c.startHeartbeatTimer()

}

//写数据包
func (c *Client) SendMessage(msg *Message) error {

    var pkg []byte
    var err error
    if c.secert == true {

        pkg, err = EnBinaryPackage(msg, c.secertKey)
        if err != nil {
            return err
        }
    } else {
        pkg, err = Encode(msg)
        if err != nil {
            return err
        }
    }
    if c.conn != nil && c.HasConnect() {
        _, err := c.conn.Write(pkg)
        if err != nil {
            log.Errorf("fd[%d] ip[%s] send err [%v]", c.fd, c.conn.RemoteAddr().String(), err)
            return err
        }
    }

    return nil
}
