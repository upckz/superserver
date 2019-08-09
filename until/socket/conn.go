package socket

import (
    "bytes"
    "encoding/binary"
    "errors"
    "github.com/koangel/grapeTimer"
    "net"
    "strconv"
    "superserver/cmd"
    "superserver/until/common"
    "superserver/until/crypto/dh64"
    log "superserver/until/czlog"
    "superserver/until/epoll"
    "sync"
)

// TCPConn represents a server connection to a TCP server, it implments Conn.
type TCPConn struct {
    server  *Server
    epoller *epoll.Epoll
    fd      int
    rawConn net.Conn
    mu      sync.Mutex
    name    string
    cache   []byte //recv cache

    sendCh chan []byte //use for send data
    readCh chan *common.MessageWrapper
    once   *sync.Once //只执行一次的锁

    hbTimeout int32
    hbTimer   int

    secertKey []byte //通讯密钥
    secert    bool   //通讯是否加密

    privateKey  uint64
    publicKey   uint64
    hasRecvFlag bool //recv flag('SW') to change to publicKey

}

// NewTCPConn returns a new server connection which has not started to
// serve requests yet.
func NewTCPConn(fd int, c net.Conn, s *Server, epoller *epoll.Epoll) *TCPConn {
    cw := &TCPConn{
        fd:          fd,
        rawConn:     c,
        cache:       make([]byte, 0),
        sendCh:      make(chan []byte, 1000),
        server:      s,
        readCh:      s.config.MsgCh,
        once:        &sync.Once{},
        hbTimeout:   s.config.HbTimeout,
        secert:      s.config.Secert,
        secertKey:   make([]byte, 0),
        hasRecvFlag: false,
        epoller:     epoller,
    }
    if !cw.secert {
        cw.hasRecvFlag = true
        cw.Connect()
    }

    cw.SetName(c.RemoteAddr().String())
    cw.StartHeartbeatTimer()
    return cw
}

func (cw *TCPConn) GetConn() net.Conn {
    return cw.rawConn
}

// GetFD returns net ID of server connection.
func (cw *TCPConn) GetFD() int {
    return cw.fd
}

// SetName sets name of server connection.
func (cw *TCPConn) SetName(name string) {
    cw.name = name
}

func (cw *TCPConn) GetName() string {
    return cw.name
}

// Write writes a message to the client.
func (cw *TCPConn) Write(message *common.Message) error {
    return asyncWrite(cw, message)
}

func (cw *TCPConn) Connect() {
    onConnect := cw.server.config.OnConnect
    if onConnect != nil {
        log.Debugf("fd[%d] [%v] connect success, total conn[%d]", cw.fd, cw.rawConn.RemoteAddr().String(), cw.server.GetTotalConnect())
        onConnect(cw.fd)
    }
}

func (cw *TCPConn) DoRecv() error {

    log.Debugf("DoRecv begin, fd[%d] <%v> ", cw.fd, cw.rawConn.RemoteAddr())
    var buf = make([]byte, 1024*64)
    n, err := cw.rawConn.Read(buf)
    if err != nil {
        log.Errorf("connection close, fd[%d] <%v> err[%v]", cw.fd, cw.rawConn.RemoteAddr(), err)
        cw.Close()
        return err
    }
    if n == 0 {
        log.Infof("connection close, fd[%d] <%v> ", cw.fd, cw.rawConn.RemoteAddr())
        cw.Close()
        return errors.New("connnect close")
    }
    cw.cache = append(cw.cache, buf[0:n]...)
    for len(cw.cache) > 0 {
        pkglen := common.ParsePacket(cw.cache)
        if pkglen < 0 {
            log.Errorf("fd[%d] [%s] recv error pkg", cw.fd, cw.rawConn.RemoteAddr())
            cw.Close()
            return err
        } else if pkglen == 0 {
            break
        } else {

            if cw.hasRecvFlag {
                msg, err := common.OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey)

                if err != nil || msg == nil {
                    log.Errorf("fd[%d] ip[%s] input fatal error, ", cw.fd, cw.rawConn.RemoteAddr().String())
                    cw.Close()
                    return err
                } else {
                    cw.ProcessDoneMsg(msg)
                }
            } else {
                err := cw.BulidEncryptionConnction(cw.cache[0:pkglen])
                if err != nil {
                    cw.Close()
                    return err
                }
            }
        }
        cw.cache = cw.cache[pkglen:]
    }
    return nil

}

func (cw *TCPConn) BulidEncryptionConnction(buff []byte) error {
    if !cw.hasRecvFlag && cw.secert {

        head := &common.PackageHead{
            Data: make([]byte, len(buff)-5),
        }

        bufReader := bytes.NewReader(buff)
        err := binary.Read(bufReader, binary.BigEndian, &head.Length)
        if err != nil {
            log.Errorf("fd[%d] %s read headlen err[%v]", cw.fd, cw.rawConn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Ack)
        if err != nil {
            log.Errorf("fd[%d] %s read head ack err[%v]", cw.fd, cw.rawConn.RemoteAddr().String(), err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Data)
        if err != nil {
            log.Errorf("fd[%d] %s read head ack err[%v]", cw.fd, cw.rawConn.RemoteAddr().String(), err.Error())
            return err
        }
        if head.Ack == 1 {
            if len(head.Data) != 2 || head.Data[0] != 'S' || head.Data[1] != 'W' {
                log.Errorf("fd[%d] %s,recv is not 'SW' error ", cw.fd, cw.rawConn.RemoteAddr().String())
                return err
            }
            cw.privateKey, cw.publicKey = dh64.KeyPair()

            wb := make([]byte, 13)
            binary.BigEndian.PutUint32(wb[:4], uint32(13))
            wb[4] = byte(2)
            binary.BigEndian.PutUint64(wb[5:], uint64(cw.publicKey))

            if _, err := cw.rawConn.Write(wb); err != nil {
                log.Errorf("fd[%d] %s, error writing data %v\n", cw.fd, cw.rawConn.RemoteAddr().String(), err.Error())
                return err
            }

        } else if head.Ack == 3 {
            clientPublicKey := uint64(binary.BigEndian.Uint64(head.Data))

            secert, err := dh64.Secret(cw.privateKey, uint64(clientPublicKey))
            if err != nil {
                log.Errorf("fd[%d] %s, DH64 secert Error:%v", cw.fd, cw.rawConn.RemoteAddr().String(), err.Error())
                cw.Close()
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

            log.Debugf("fd[%d] connectHandler DH64 serverPublicKey[%v] clientPublicKey[%v] secert[%v] secertStr[%v] ip[%v] connect",
                cw.fd, cw.publicKey, clientPublicKey, secert, secertStr, cw.rawConn.RemoteAddr().String())

            log.Infof("fd[%d] [%v] connect success, total conn[%d]", cw.fd, cw.rawConn.RemoteAddr().String(), cw.server.GetTotalConnect())
            sw := keyBuffer.Bytes()
            cw.secertKey = make([]byte, len(sw))
            copy(cw.secertKey, sw)
            cw.hasRecvFlag = true
            cw.Connect()
        } else {
            log.Errorf("fd[%d] %s, ack", cw.fd, cw.rawConn.RemoteAddr().String())
            cw.Close()
            return err
        }
    }

    return nil
}

func (cw *TCPConn) ProcessDoneMsg(msg *common.Message) int {

    log.Warnf("fd[%d]  ip[%s] recv message %v\n", cw.fd, cw.rawConn.RemoteAddr().String(), msg)
    if msg.GetCmd() == cmd.MsgHeartbeat {
        cw.SendHeartbeatMsg(msg.GetUid())
    }
    cw.StartHeartbeatTimer()

    // mw := &common.MessageWrapper{
    //     Msg:      msg,
    //     ConnID:   cw.fd,
    //     SourceIP: cw.rawConn.RemoteAddr().String(),
    // }

    // if len(cw.readCh) > 1000 {
    //     log.Errorf("readch max 1000 error[%d]", len(cw.readCh))
    // }
    // // cw.readCh <- mw

    return 0

}

// Close gracefully closes the client connection.
func (cw *TCPConn) Close() {
    cw.once.Do(func() {
        log.Infof("conn close, < %v>\n", cw.rawConn.RemoteAddr())
        // close net.Conn, any blocked read or write operation will be unblocked and
        // return errors.
        onClose := cw.server.config.OnClose
        if onClose != nil {
            onClose(cw.fd)
        }
        cw.StopAllTimer()
        if err := cw.epoller.Remove(cw.GetFD()); err != nil {
            log.Errorf("<%v> failed to remove %v", cw.GetConn().RemoteAddr(), err)
        }
        cw.server.RemovConn(cw.fd)
        cw.rawConn.Close()
        close(cw.sendCh)
        cw.hasRecvFlag = false
    })
}

func (cw *TCPConn) StopAllTimer() {
    cw.StopHeartbeatTimer()
}

func (cw *TCPConn) StartHeartbeatTimer() {
    cw.StopHeartbeatTimer()
    cw.hbTimer = grapeTimer.NewTickerOnce(int(cw.hbTimeout*1000), cw.ProcessTimeOut, HEAET_TIMER_OUT)
}

func (cw *TCPConn) StopHeartbeatTimer() {
    grapeTimer.StopTimer(cw.hbTimer)
}

func (cw *TCPConn) ProcessTimeOut(timeType int) {

    switch timeType {
    case HEAET_TIMER_OUT:
        log.Errorf("fd[%d] ip[%s] heart time[>%d] out", cw.fd, cw.rawConn.RemoteAddr().String(), cw.hbTimeout)
        cw.Close()
    default:
    }
}

func (cw *TCPConn) SendHeartbeatMsg(uid int32) {
    msg := common.NewMessage(cmd.MsgHeartbeat, uid, make([]byte, 0))
    err := cw.Write(msg)
    if err != nil {
        log.Errorf("fd[%d] uid[%d] ip[%s] send heart error", cw.fd, uid, cw.rawConn.RemoteAddr().String())
        return
    } else {
        log.Debugf("fd[%d] [uid[%d] ip[%s] send pkg[%v]", cw.fd, uid, cw.rawConn.RemoteAddr().String(), msg)
    }

}

func asyncWrite(cw *TCPConn, m *common.Message) error {
    defer func() error {
        if p := recover(); p != nil {
            log.Errorln(common.GlobalPanicLog(p))
            return ErrServerClosed
        }
        return nil
    }()

    var pkt []byte
    var err error
    if cw.secert == true {
        pkt, err = common.EnBinaryPackage(m, cw.secertKey)
        if err != nil {
            return err
        }
    } else {
        pkt, err = common.Encode(m)
        if err != nil {
            return err
        }
    }
    _, err = cw.rawConn.Write(pkt)
    if err != nil {
        log.Errorf("error writing data %v\n", err)
        cw.Close()
        return err
    }
    return nil
}
