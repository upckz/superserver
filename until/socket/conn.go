package socket

import (
    "net"
    "sync"
    "bytes"
    "strconv"
    "superserver/until/common"
    "encoding/binary"
    log "superserver/until/czlog"
    "superserver/until/crypto/dh64"
    "github.com/koangel/grapeTimer"
    "superserver/cmd"
)

// TCPConn represents a server connection to a TCP server, it implments Conn.
type TCPConn struct {
    server  *Server
    netid   int
    rawConn net.Conn
    mu      sync.Mutex
    name    string
    cache   []byte //recv cache


    sendCh  chan []byte                    //use for send data
    readCh  chan *MessageWrapper
    once    *sync.Once                     //只执行一次的锁

    hbTimeout  int32
    hbTimer    int

    secertKey  []byte //通讯密钥
    secert     bool   //通讯是否加密

    privateKey  uint64
    publicKey   uint64
    hasRecvFlag  bool   //recv flag('SW') to change to publicKey

}

// NewTCPConn returns a new server connection which has not started to
// serve requests yet.
func NewTCPConn(id int, c net.Conn, s *Server) *TCPConn {
    cw := &TCPConn{
        netid:   id,
        rawConn: c,
        cache:   make([]byte, 0),
        sendCh:  make(chan []byte, 1000),
        server:  s,
        readCh:  s.config.MsgCh,
        once:    &sync.Once{},
        hbTimeout:  s.config.HbTimeout,
        secert:     s.config.Secert,
        secertKey:  make([]byte, 0),
        hasRecvFlag:    false,
    }

    cw.SetName(c.RemoteAddr().String())
    cw.StartHeartbeatTimer()
    return cw
}

// GetNetID returns net ID of server connection.
func (cw *TCPConn) GetNetID() int {
    return cw.netid
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


func (cw *TCPConn)Connect() {
    onConnect := cw.server.config.OnConnect
    if onConnect != nil {
        onConnect(cw.netid)
    }
}



func (cw *TCPConn)Recv(epoller *epoll, conn net.Conn) {

    var buf = make([]byte,1024*64)
    n , err :=  conn.Read(buf)
    if err != nil {
        if n == 0 {
            log.Infof("connection close, connId[%d] <%v> ",cw.netid, conn.RemoteAddr())
        } else {
            log.Errorf("connection close, connId[%d] <%v> err[%v]",cw.netid, conn.RemoteAddr(), err)
        }
        
        if err := epoller.Remove(conn); err != nil {
            log.Errorf("connId[%d] <%v> failed to remove %v",cw.netid,  conn.RemoteAddr(), err)
        }
        cw.Close()
      
    } else {
        cw.cache = append(cw.cache, buf[0:n]...)
        for len(cw.cache) > 0 {
            pkglen := ParsePacket(cw.cache)
            if pkglen < 0 {
                log.Errorf("connId[%d] [%s] recv error pkg",cw.netid, conn.RemoteAddr(),)   
                cw.Close()
                break
            } else if pkglen == 0 {
                continue
            } else {
                if (cw.secert && cw.hasRecvFlag) || !cw.secert {
                    msg, err := OnPacketComplete(cw.cache[0:pkglen], cw.secert, cw.secertKey) 

                    if err != nil || msg == nil{
                        log.Errorf("connID[%d] ip[%s] input fatal error, ", cw.netid, cw.rawConn.RemoteAddr().String(),)
                        cw.Close()
                        break
                    } else {
                         cw.ProcessDoneMsg(msg) 
                    }
                } else {
                    err := cw.BulidEncryptionConnction(cw.cache[0:pkglen])
                    if err != nil {
                        cw.Close()
                        break
                    }
                }
                cw.cache = cw.cache[pkglen:]
            }
         }
    }
}

func (cw *TCPConn)BulidEncryptionConnction(buff []byte) error {
    if !cw.hasRecvFlag  {

        head := &PackageHead{
            data: make([]byte, len(buff)-5),
        }

        bufReader := bytes.NewReader(buff)
        err := binary.Read(bufReader, binary.BigEndian, &head.length)
        if err != nil {
            log.Errorf("connId[%d] %s read headlen err[%v]", cw.netid, cw.rawConn.RemoteAddr().String(),err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.ack)
        if err != nil {
            log.Errorf("connId[%d] %s read head ack err[%v]", cw.netid, cw.rawConn.RemoteAddr().String(),err.Error())
            return err
        }
        err = binary.Read(bufReader, binary.BigEndian, &head.data)
        if err != nil {
            log.Errorf("connId[%d] %s read head ack err[%v]", cw.netid, cw.rawConn.RemoteAddr().String(), err.Error())
            return err
        }
        if head.ack == 1 {
            if len(head.data) != 2 || head.data[0] != 'S' || head.data[1] != 'W' {
                log.Errorf("connId[%d] %s,recv is not 'SW' error ",cw.netid,  cw.rawConn.RemoteAddr().String())
                return err
            }
            cw.privateKey, cw.publicKey = dh64.KeyPair()

            wb := make([]byte, 13)
            binary.BigEndian.PutUint32(wb[:4], uint32(13))
            wb[4] = byte(2)
            binary.BigEndian.PutUint64(wb[5:], uint64(cw.publicKey))

            if _, err := cw.rawConn.Write(wb); err != nil {
                log.Errorf("connId[%d] %s, error writing data %v\n",cw.netid,  cw.rawConn.RemoteAddr().String(), err.Error())
                return err
            }

        } else if head.ack == 3 {
            clientPublicKey :=  uint64(binary.BigEndian.Uint64(head.data))

            secert, err := dh64.Secret(cw.privateKey, uint64(clientPublicKey))
            if err != nil {
                log.Errorf("connId[%d] %s, DH64 secert Error:%v",cw.netid,  cw.rawConn.RemoteAddr().String(),err.Error())
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

            log.Debugf("connId[%d] connectHandler DH64 serverPublicKey[%v] clientPublicKey[%v] secert[%v] secertStr[%v] ip[%v] connect",
                    cw.netid, cw.publicKey,clientPublicKey,secert,secertStr, cw.rawConn.RemoteAddr().String())

            log.Infof("connId[%d] [%v] connect success, total conn[%d]", cw.netid, cw.rawConn.RemoteAddr().String(), cw.server.GetTotalConnect())
            sw := keyBuffer.Bytes()
            cw.secertKey = make([]byte, len(sw))
            copy(cw.secertKey, sw)
            cw.hasRecvFlag = true
            cw.Connect()
        }
    }
    return nil
}



func (cw *TCPConn)ProcessDoneMsg(msg *Message) int {
   
    log.Warnf("connID[%d]  ip[%s] recv message %v\n", cw.netid, cw.rawConn.RemoteAddr().String(), msg)
    if msg.GetCmd() == cmd.MsgHeartbeat {
        cw.SendHeartbeatMsg(msg.GetUid())
    }
    // mw := &MessageWrapper {
    //     Msg:        msg,
    //     ConnID:     cw.netid,
    //     SourceIP:   cw.rawConn.RemoteAddr().String(),
    // }
    //收到一个包算一次心跳
    //写操作可能处理不过来，需要等待，暂时停止心跳检测
    // cw.StopHeartbeatTimer()
    // cw.readCh <- mw  
    // cw.StartHeartbeatTimer()
    
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
            onClose(cw.netid)
        }
        cw.StopAllTimer()
        cw.rawConn.Close()
        close(cw.sendCh)
        cw.server.conns.Remove(cw.netid)
    })
}

func (cw *TCPConn)StopAllTimer() {
    cw.StopHeartbeatTimer()
}

func (cw *TCPConn)StartHeartbeatTimer() {
    cw.StopHeartbeatTimer()
    cw.hbTimer = grapeTimer.NewTickerLoop(int(cw.hbTimeout*1000), -1, cw.ProcessTimeOut, HEAET_TIMER_OUT)
}

func (cw *TCPConn)StopHeartbeatTimer(){
    grapeTimer.StopTimer(cw.hbTimer)
}

func (cw *TCPConn)ProcessTimeOut(timeType int) {

    switch(timeType) {
    case HEAET_TIMER_OUT:
        log.Errorf("connID[%d] ip[%s] heart time out", cw.netid, cw.rawConn.RemoteAddr().String())
        cw.Close()
    default:
    }
}

func (cw *TCPConn)SendHeartbeatMsg( uid int32) {
    msg := NewMessage(cmd.MsgHeartbeat, uid, make([]byte, 0))
    err := cw.Write(msg)
    if err != nil {
        log.Errorf("connID[%d] uid[%d] ip[%s] send heart error", cw.netid, uid, cw.rawConn.RemoteAddr().String())
        return
    } else {
        log.Debugf("connID[%d] [uid[%d] ip[%s] send pkg[%v]", cw.netid, uid, cw.rawConn.RemoteAddr().String(), msg)
    }
   
}


func asyncWrite(cw *TCPConn, m *Message) error {
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
        pkt, err = EnBinaryPackage(m, cw.secertKey)
        if err != nil {
            return err 
        }
    }  else {
        pkt ,err = Encode(m)
        if err != nil {
            return err
        }
    }
    _, err =  cw.rawConn.Write(pkt)
    if err != nil {
        log.Errorf("error writing data %v\n", err)
        cw.Close()
        return err
    }
    return nil
}


