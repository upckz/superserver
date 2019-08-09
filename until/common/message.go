package common

import "fmt"

type Message struct {
    cmd  uint32
    uid  int32
    data []byte
}

func NewMessage(cmd uint32, uid int32, data []byte) *Message {
    msg := &Message{
        cmd:  cmd,
        uid:  uid,
        data: data,
    }
    return msg
}

func (msg *Message) GetData() []byte {
    return msg.data
}

func (msg *Message) GetCmd() uint32 {
    return msg.cmd
}

func (msg *Message) GetUid() int32 {
    return msg.uid
}

func (msg *Message) String() string {
    return fmt.Sprintf("cmd:0x%04x UID:%d  datalen:%d", msg.GetCmd(), msg.GetUid(), len(msg.GetData()))
}

type MessageWrapper struct {
    Msg      *Message
    ConnID   int
    SourceIP string
}

type ResponseWrapper struct {
    Msg    *Message
    ConnID int
    UserID int32
}

type PackageHead struct {
    Length uint32
    Ack    uint8 // 1 client to send 'SW', 2 server send to client publicKey, 3 client send to server clientPublicKey
    Data   []byte
}
