package socket

import (
    "bytes"
    "encoding/binary"
    "errors"
    "net"
    "strconv"
    "superserver/cmd"
    //"superserver/until/common"
    "superserver/until/crypto/dh64"
    "superserver/until/log"
    "superserver/until/timingwheel"
    "sync"
    "time"
)

// TCPConn represents a server connection to a TCP server, it implments Conn.
type TCPConn struct {
    server  *Server
    epoller *epoll
    fd      int
    rawConn net.Conn
    mu      sync.Mutex
    name    string
    cache   []byte //recv cache

    sendCh chan []byte //use for send data
    readCh chan *MessageWrapper
    once   *sync.Once //只执行一次的锁

    secertKey []byte //通讯密钥
    secert    bool   //通讯是否加密

    privateKey  uint64
    publicKey   uint64
    hasRecvFlag bool //recv flag('SW') to change to publicKey

    hbTimeout  time.Duration
    hbTimer    *timingwheel.Timer
    timerWheel *timingwheel.TimingWheel
}

// NewTCPConn returns a new server connection which has not started to
// serve requests yet.
func NewTCPConn(fd int, c net.Conn, s *Server, epoller *epoll) *TCPConn {
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
        timerWheel:  s.timerWheel,
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
func (cw *TCPConn) Write(message *Message) error {
    return asyncWrite(cw, message)
}

func (cw *TCPConn) Connect() {
    onConnect := cw.server.config.OnConnect
    if onConnect != nil {
        log.WithFields(log.Fields{
            "total conn": cw.server.GetTotalConnect(),
            "fd":         cw.fd,
            "ip":         cw.rawConn.RemoteAddr().String(),
        }).Debug(" connect success")
        onConnect(cw.fd)
    }
}

func (cw *TCPConn) DoRecv() error {

    var buf = make([]byte, 1024*64)
    n, err := cw.rawConn.Read(buf)
    if err != nil {
        log.WithFields(log.Fields{
            "fd":  cw.fd,
            "err": err,
            "ip":  cw.rawConn.RemoteAddr().String(),
        }).Error("connection close")
        cw.Close()
        return err
    }
    if n == 0 {
        log.WithFields(log.Fields{
            "fd": cw.fd,
            "ip": cw.rawConn.RemoteAddr().String(),
        }).Info("connection close")
        cw.Close()
        return errors.New("connnect close")
    }
    cw.cache = append(cw.cache, buf[0:n]...)
    for len(cw.cache) > 0 {
        pkglen := ParsePacket(cw.cache)
        if pkglen < 0 {
            log.WithFields(log.Fields{
                "fd": cw.fd,
                "ip": cw.rawConn.RemoteAddr().String(),
            }).Error("recv error pkg")

            cw.Close()
            return err
        } else if pkglen == 0 {
            break
        } else {

            if cw.hasRecvFlag {
                msg, err := OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey)

                if err != nil || msg == nil {
                    log.WithFields(log.Fields{
                        "fd": cw.fd,
                        "ip": cw.rawConn.RemoteAddr().String(),
                    }).Error("input fatal error")
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

        head := &PackageHead{
            Data: make([]byte, len(buff)-5),
        }

        bufReader := bytes.NewReader(buff)
        err := binary.Read(bufReader, binary.BigEndian, &head.Length)
        if err != nil {
            log.WithFields(log.Fields{
                "fd":  cw.fd,
                "err": err.Error(),
                "ip":  cw.rawConn.RemoteAddr().String(),
            }).Error("read headlen error")

            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Ack)
        if err != nil {
            log.WithFields(log.Fields{
                "fd":  cw.fd,
                "err": err.Error(),
                "ip":  cw.rawConn.RemoteAddr().String(),
            }).Error(" read head ack error")
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.Data)
        if err != nil {
            log.WithFields(log.Fields{
                "fd":  cw.fd,
                "err": err.Error(),
                "ip":  cw.rawConn.RemoteAddr().String(),
            }).Error("read head ack err")
            return err
        }
        if head.Ack == 1 {
            if len(head.Data) != 2 || head.Data[0] != 'S' || head.Data[1] != 'W' {
                log.WithFields(log.Fields{
                    "fd": cw.fd,
                    "ip": cw.rawConn.RemoteAddr().String(),
                }).Error("recv is not 'SW' error")
                return err
            }
            cw.privateKey, cw.publicKey = dh64.KeyPair()

            wb := make([]byte, 13)
            binary.BigEndian.PutUint32(wb[:4], uint32(13))
            wb[4] = byte(2)
            binary.BigEndian.PutUint64(wb[5:], uint64(cw.publicKey))

            if _, err := cw.rawConn.Write(wb); err != nil {
                log.WithFields(log.Fields{
                    "fd":  cw.fd,
                    "err": err.Error(),
                    "ip":  cw.rawConn.RemoteAddr().String(),
                }).Error("writing data error")
                return err
            }

        } else if head.Ack == 3 {
            clientPublicKey := uint64(binary.BigEndian.Uint64(head.Data))

            secert, err := dh64.Secret(cw.privateKey, uint64(clientPublicKey))
            if err != nil {
                log.WithFields(log.Fields{
                    "fd":  cw.fd,
                    "err": err.Error(),
                    "ip":  cw.rawConn.RemoteAddr().String(),
                }).Error("DH64 secert Error")
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
            log.WithFields(log.Fields{
                "fd":              cw.fd,
                "publicKey":       cw.publicKey,
                "clientPublicKey": clientPublicKey,
                "secert":          secert,
                "secertStr":       secertStr,
                "ip":              cw.rawConn.RemoteAddr().String(),
                "total conn":      cw.server.GetTotalConnect(),
            }).Info("connect success")

            sw := keyBuffer.Bytes()
            cw.secertKey = make([]byte, len(sw))
            copy(cw.secertKey, sw)
            cw.hasRecvFlag = true
            cw.Connect()
        } else {
            log.WithFields(log.Fields{
                "fd": cw.fd,
                "ip": cw.rawConn.RemoteAddr().String(),
            }).Error("ack Error")
            cw.Close()
            return err
        }
    }

    return nil
}

func (cw *TCPConn) ProcessDoneMsg(msg *Message) int {

    log.WithFields(log.Fields{
        "fd": cw.fd,
        "ip": cw.rawConn.RemoteAddr().String(),
    }).Debug(msg)

    if msg.GetCmd() == cmd.MsgHeartbeat {
        cw.SendHeartbeatMsg(msg.GetUid())
    }
    cw.StartHeartbeatTimer()

    mw := &MessageWrapper{
        Msg:      msg,
        ConnID:   cw.fd,
        SourceIP: cw.rawConn.RemoteAddr().String(),
    }

    cw.readCh <- mw
    return 0

}

// Close gracefully closes the client connection.
func (cw *TCPConn) Close() {
    cw.once.Do(func() {
        log.WithFields(log.Fields{
            "fd": cw.fd,
            "ip": cw.rawConn.RemoteAddr().String(),
        }).Info("conn close")

        // close net.Conn, any blocked read or write operation will be unblocked and
        // return errors.
        onClose := cw.server.config.OnClose
        if onClose != nil {
            onClose(cw.fd)
        }
        cw.StopAllTimer()
        if err := cw.epoller.Remove(cw.GetFD()); err != nil {
            log.WithFields(log.Fields{
                "fd":  cw.fd,
                "err": err,
                "ip":  cw.rawConn.RemoteAddr().String(),
            }).Error("failed to remove")

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
    cw.hbTimer = cw.timerWheel.AfterFunc(cw.hbTimeout, cw.ProcessTimeOut, HEAET_TIMER_OUT)
}

func (cw *TCPConn) StopHeartbeatTimer() {
    if cw.hbTimer != nil {
        cw.hbTimer.Stop()
    }
    cw.hbTimer = nil

}

func (cw *TCPConn) ProcessTimeOut(timeType int) {

    switch timeType {
    case HEAET_TIMER_OUT:
        log.WithFields(log.Fields{
            "fd":            cw.fd,
            "heartTimeOut ": cw.hbTimeout,
            "ip":            cw.rawConn.RemoteAddr().String(),
        }).Error("heartTimeOut")
        cw.Close()
    default:
    }
}

func (cw *TCPConn) SendHeartbeatMsg(uid int32) {
    msg := NewMessage(cmd.MsgHeartbeat, uid, make([]byte, 0))
    err := cw.Write(msg)
    if err != nil {
        log.WithFields(log.Fields{
            "uid": uid,
            "fd":  cw.fd,
            "ip":  cw.rawConn.RemoteAddr().String(),
        }).Error("send heart error")

        return
    } else {
        log.WithFields(log.Fields{
            "uid":  uid,
            "fd":   cw.fd,
            "ip":   cw.rawConn.RemoteAddr().String(),
            "send": "success",
        }).Debug(msg)
    }

}

func asyncWrite(cw *TCPConn, m *Message) error {
    defer func() error {
        if p := recover(); p != nil {
            // log.Errorln(common.GlobalPanicLog(p))
            log.WithFields(log.Fields{
                "fd": cw.fd,
                "ip": cw.rawConn.RemoteAddr().String(),
            }).Panic(p)
            return ErrServerClosed
        }
        return nil
    }()

    var pkt []byte
    var err error
    if cw.secert == true {
        pkt, err = EnBinaryPackage(m, cw.secertKey)
        if err != nil {
            return err
        }
    } else {
        pkt, err = Encode(m)
        if err != nil {
            return err
        }
    }
    _, err = cw.rawConn.Write(pkt)
    if err != nil {
        log.WithFields(log.Fields{
            "fd":  cw.fd,
            "err": err,
            "ip":  cw.rawConn.RemoteAddr().String(),
        }).Error("error writing data")
        cw.Close()
        return err
    }
    return nil
}
