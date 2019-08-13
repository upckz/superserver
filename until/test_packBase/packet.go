package test

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
)

var m_SendByteMap = [256]byte{
    0x29, 0x23, 0xBE, 0x84, 0xE1, 0x6C, 0xD6, 0xAE, 0x52, 0x90, 0x49, 0xF1, 0xBB, 0xE9, 0xEB, 0xB3,
    0xA6, 0xDB, 0x3C, 0x87, 0x0C, 0x3E, 0x99, 0x24, 0x5E, 0x0D, 0x1C, 0x06, 0xB7, 0x47, 0xDE, 0x12,
    0x4D, 0xC8, 0x43, 0x8B, 0x1F, 0x03, 0x5A, 0x7D, 0x09, 0x38, 0x25, 0x5D, 0xD4, 0xCB, 0xFC, 0x96,
    0xF5, 0x45, 0x3B, 0x13, 0x89, 0x0A, 0x32, 0x20, 0x9A, 0x50, 0xEE, 0x40, 0x78, 0x36, 0xFD, 0xF6,
    0x9E, 0xDC, 0xAD, 0x4F, 0x14, 0xF2, 0x44, 0x66, 0xD0, 0x6B, 0xC4, 0x30, 0xA1, 0x22, 0x91, 0x9D,
    0xDA, 0xB0, 0xCA, 0x02, 0xB9, 0x72, 0x2C, 0x80, 0x7E, 0xC5, 0xD5, 0xB2, 0xEA, 0xC9, 0xCC, 0x53,
    0xBF, 0x67, 0x2D, 0x8E, 0x83, 0xEF, 0x57, 0x61, 0xFF, 0x69, 0x8F, 0xCD, 0xD1, 0x1E, 0x9C, 0x16,
    0xE6, 0x1D, 0xF0, 0x4A, 0x77, 0xD7, 0xE8, 0x39, 0x33, 0x74, 0xF4, 0x9F, 0xA4, 0x59, 0x35, 0xCF,
    0xD3, 0x48, 0x75, 0xD9, 0x2A, 0xE5, 0xC0, 0xF7, 0x2B, 0x81, 0x0E, 0x5F, 0x00, 0x8D, 0x7B, 0x05,
    0x15, 0x07, 0x82, 0x18, 0x70, 0x92, 0x64, 0x54, 0xCE, 0xB1, 0x85, 0xF8, 0x46, 0x6A, 0x04, 0x73,
    0x2F, 0x68, 0x76, 0xFA, 0x11, 0x88, 0x79, 0xFE, 0xD8, 0x28, 0x0B, 0x60, 0x3D, 0x97, 0x27, 0x8A,
    0xC2, 0x08, 0xA5, 0xC1, 0x8C, 0xA9, 0x95, 0x9B, 0xA8, 0xA7, 0x86, 0xB5, 0xE7, 0x55, 0x4E, 0x71,
    0xE2, 0xB4, 0x65, 0x7A, 0x63, 0x26, 0xDF, 0x6D, 0x62, 0xE0, 0x34, 0x3F, 0xE3, 0x41, 0x0F, 0x1B,
    0xF3, 0xA0, 0x7F, 0xAA, 0x5B, 0xB8, 0x3A, 0x10, 0x4C, 0xEC, 0x31, 0x42, 0x7C, 0xE4, 0x21, 0x93,
    0xAF, 0x6F, 0x01, 0x17, 0x56, 0xC6, 0xF9, 0x37, 0xBD, 0x6E, 0x5C, 0xC3, 0xA3, 0x98, 0xC7, 0xB6,
    0x51, 0x19, 0x2E, 0xBC, 0x94, 0x4B, 0x58, 0xD2, 0xAC, 0x1A, 0xA2, 0xED, 0xFB, 0xDD, 0xBA, 0xAB,
}

var m_RecvByteMap = [256]byte{
    0x8C, 0xE2, 0x53, 0x25, 0x9E, 0x8F, 0x1B, 0x91, 0xB1, 0x28, 0x35, 0xAA, 0x14, 0x19, 0x8A, 0xCE,
    0xD7, 0xA4, 0x1F, 0x33, 0x44, 0x90, 0x6F, 0xE3, 0x93, 0xF1, 0xF9, 0xCF, 0x1A, 0x71, 0x6D, 0x24,
    0x37, 0xDE, 0x4D, 0x01, 0x17, 0x2A, 0xC5, 0xAE, 0xA9, 0x00, 0x84, 0x88, 0x56, 0x62, 0xF2, 0xA0,
    0x4B, 0xDA, 0x36, 0x78, 0xCA, 0x7E, 0x3D, 0xE7, 0x29, 0x77, 0xD6, 0x32, 0x12, 0xAC, 0x15, 0xCB,
    0x3B, 0xCD, 0xDB, 0x22, 0x46, 0x31, 0x9C, 0x1D, 0x81, 0x0A, 0x73, 0xF5, 0xD8, 0x20, 0xBE, 0x43,
    0x39, 0xF0, 0x08, 0x5F, 0x97, 0xBD, 0xE4, 0x66, 0xF6, 0x7D, 0x26, 0xD4, 0xEA, 0x2B, 0x18, 0x8B,
    0xAB, 0x67, 0xC8, 0xC4, 0x96, 0xC2, 0x47, 0x61, 0xA1, 0x69, 0x9D, 0x49, 0x05, 0xC7, 0xE9, 0xE1,
    0x94, 0xBF, 0x55, 0x9F, 0x79, 0x82, 0xA2, 0x74, 0x3C, 0xA6, 0xC3, 0x8E, 0xDC, 0x27, 0x58, 0xD2,
    0x57, 0x89, 0x92, 0x64, 0x03, 0x9A, 0xBA, 0x13, 0xA5, 0x34, 0xAF, 0x23, 0xB4, 0x8D, 0x63, 0x6A,
    0x09, 0x4E, 0x95, 0xDF, 0xF4, 0xB6, 0x2F, 0xAD, 0xED, 0x16, 0x38, 0xB7, 0x6E, 0x4F, 0x40, 0x7B,
    0xD1, 0x4C, 0xFA, 0xEC, 0x7C, 0xB2, 0x10, 0xB9, 0xB8, 0xB5, 0xD3, 0xFF, 0xF8, 0x42, 0x07, 0xE0,
    0x51, 0x99, 0x5B, 0x0F, 0xC1, 0xBB, 0xEF, 0x1C, 0xD5, 0x54, 0xFE, 0x0C, 0xF3, 0xE8, 0x02, 0x60,
    0x86, 0xB3, 0xB0, 0xEB, 0x4A, 0x59, 0xE5, 0xEE, 0x21, 0x5D, 0x52, 0x2D, 0x5E, 0x6B, 0x98, 0x7F,
    0x48, 0x6C, 0xF7, 0x80, 0x2C, 0x5A, 0x06, 0x75, 0xA8, 0x83, 0x50, 0x11, 0x41, 0xFD, 0x1E, 0xC6,
    0xC9, 0x04, 0xC0, 0xCC, 0xDD, 0x85, 0x70, 0xBC, 0x76, 0x0D, 0x5C, 0x0E, 0xD9, 0xFB, 0x3A, 0x65,
    0x72, 0x0B, 0x45, 0xD0, 0x7A, 0x30, 0x3F, 0x87, 0x9B, 0xE6, 0xA3, 0xFC, 0x2E, 0x3E, 0xA7, 0x68,
}

const (
    ICHAT_TCP_MIN_BUFFER     int32 = 1024
    ICHAT_TCP_DEFAULT_BUFFER int32 = 65536
    ICHAT_TCP_MAX_BUFFER     int32 = 1024 * 64
    ICHAT_UDP_DEFAULT_BUFFER int32 = 1024 * 64
    MAX_PACKET_LEN           int32 = 1024 * 64
    PACKET_BUFFER_SIZE       int32 = 1024 * 64
    PACKET_HEADER_SIZE       int32 = 11
)

type PkgHeader struct {
    length int32
    check  []byte
    cmd    int32
    code   byte
}

func NewPkgHeader() *PkgHeader {
    p := &PkgHeader{
        length: 0,
        check:  make([]byte, 2),
        cmd:    0,
        code:   0,
    }
    return p
}

func (c *PkgHeader) Copy(data []byte) error {
    l := int32(len(data))
    if l < PACKET_HEADER_SIZE {
        return errors.New("len too small")
    }

    LenBuf := data[0:4]
    bufReader := bytes.NewReader(LenBuf)
    var headerLen int32 = 0
    err := binary.Read(bufReader, binary.BigEndian, &headerLen)
    if err != nil {
        return err

    }
    c.length = headerLen
    c.check[0] = data[4]
    c.check[1] = data[5]

    cmdBuf := data[6:10]
    bufReader = bytes.NewReader(cmdBuf)
    var cmd int32 = 0
    err = binary.Read(bufReader, binary.BigEndian, &cmd)
    if err != nil {
        return err

    }
    c.cmd = cmd
    c.code = data[10]
    //fmt.Printf("---------------cmd[0x%04x] c.code[%d] data[%v]\n", cmd,c.code,data)
    return nil
}

func (c *PkgHeader) Serialize() []byte {

    s1 := make([]byte, 0)
    length := bytes.NewBuffer(s1)
    binary.Write(length, binary.BigEndian, int32(c.length))

    cmd := bytes.NewBuffer(s1)
    binary.Write(cmd, binary.BigEndian, int32(c.cmd))
    code := make([]byte, 1)
    code[0] = c.code

    buf := make([][]byte, 1+c.length)
    buf = append(buf, length.Bytes())
    buf = append(buf, c.check)
    buf = append(buf, cmd.Bytes())
    buf = append(buf, code)

    return bytes.Join(buf, []byte(""))
}

type CPacketBase struct {
    m_strBuf  []byte
    m_nHead   *PkgHeader
    m_nBufPos int32
}

func NewCPacketBase() *CPacketBase {
    p := &CPacketBase{
        m_strBuf:  make([]byte, 0),
        m_nHead:   NewPkgHeader(),
        m_nBufPos: 0,
    }
    p.m_nHead.check[0] = 'L'
    p.m_nHead.check[1] = 'W'
    return p
}

func (p *CPacketBase) GetCmd() int32 {
    return p.m_nHead.cmd
}

func (p *CPacketBase) SetCmd(cmd int32) {
    p.m_nHead.cmd = cmd
}

func (c *CPacketBase) begin(cmdType int32) {
    c.m_nHead.cmd = cmdType
    c.m_nHead.check[0] = 'L'
    c.m_nHead.check[1] = 'W'
    c.m_nHead.code = 0
    c.m_nBufPos = 0
    c.m_strBuf = make([]byte, 0)
}

func (c *CPacketBase) read(size int32) ([]byte, error) {
    packetSize := int32(len(c.m_strBuf))
    // fmt.Printf("size[%d] c.m_nBufPos[%d] packetSize[%d]\n", size, c.m_nBufPos,packetSize)
    if size+c.m_nBufPos > packetSize || size+c.m_nBufPos > PACKET_BUFFER_SIZE {
        return nil, errors.New("size is too long or too short")
    }
    tSize := c.m_nBufPos + size
    tmp := c.m_strBuf[c.m_nBufPos:tSize]
    c.m_nBufPos = tSize
    return tmp, nil
}

func (c *CPacketBase) write(data []byte) error {
    l := int32(len(data))
    if l <= 0 || l > PACKET_BUFFER_SIZE {
        return errors.New("len is too long or too short")
    }
    c.m_strBuf = append(c.m_strBuf, data...)
    c.m_nBufPos = c.m_nBufPos + l
    return nil
}

func (c *CPacketBase) Copy(data []byte) error {
    l := int32(len(data))
    if l < PACKET_HEADER_SIZE {
        return errors.New("len too small")
    }
    err := c.m_nHead.Copy(data)
    if err != nil {
        return err
    }
    return c.write(data[11:])
}

func (c *CPacketBase) Serialize() []byte {

    data := c.m_nHead.Serialize()
    value := append(data, c.m_strBuf...)
    return value
}

type NETOutputPacket struct {
    cBase        *CPacketBase
    m_bCheckCode bool
}

func NewNETOutputPacket() *NETOutputPacket {
    p := &NETOutputPacket{
        cBase:        NewCPacketBase(),
        m_bCheckCode: false,
    }

    return p
}

func (p *NETOutputPacket) Begin(cmd int32) {
    p.cBase.SetCmd(cmd)
    p.m_bCheckCode = false
}

/*func (p *NETOutputPacket)WriteInt(value int) error{
    tmp := bytes.NewBuffer(make([]byte,0))
    binary.Write(tmp, binary.BigEndian, int(value))
    return p.cBase.write(tmp.Bytes())
}*/

func (p *NETOutputPacket) WriteInt32(value int32) error {
    tmp := bytes.NewBuffer(make([]byte, 0))
    binary.Write(tmp, binary.BigEndian, int32(value))
    return p.cBase.write(tmp.Bytes())
}

func (p *NETOutputPacket) WriteInt64(value int64) error {
    tmp := bytes.NewBuffer(make([]byte, 0))
    binary.Write(tmp, binary.BigEndian, int64(value))
    return p.cBase.write(tmp.Bytes())
}

func (p *NETOutputPacket) WriteShort(value int16) error {
    tmp := bytes.NewBuffer(make([]byte, 0))
    binary.Write(tmp, binary.BigEndian, int16(value))
    return p.cBase.write(tmp.Bytes())
}

func (p *NETInputPacket) WriteByte(value byte) error {
    tmp := make([]byte, 1)
    tmp[0] = value
    return p.cBase.write(tmp)
}

func (p *NETOutputPacket) WriteString(value string) error {
    l := int32(len(value))
    err := p.WriteInt32(l)
    if err != nil {
        return err
    }
    data := []byte(value)
    return p.cBase.write(data)

}

func (p *NETOutputPacket) WriteBinary(value []byte) error {
    l := int32(len(value))
    err := p.WriteInt32(l)
    if err != nil {
        return err
    }
    return p.cBase.write(value)
}

func (p *NETOutputPacket) IsWritecbCheckCode() bool {
    return p.m_bCheckCode
}

func (p *NETOutputPacket) End() {
    p.m_bCheckCode = true
    p.cBase.m_nHead.length = p.cBase.m_nBufPos + PACKET_HEADER_SIZE
}

func (p *NETOutputPacket) GetBody() []byte {
    return p.cBase.m_strBuf
}

func (p *NETOutputPacket) WritecbCheckCode(value byte) {
    p.cBase.m_nHead.code = value
}

func (p *NETOutputPacket) Serialize() []byte {
    return p.cBase.Serialize()
}

type NETInputPacket struct {
    cBase *CPacketBase
}

func NewNETInputPacket(data []byte) *NETInputPacket {
    c := &NETInputPacket{
        cBase: NewCPacketBase(),
    }
    err := c.cBase.Copy(data)
    if err != nil {
        return nil
    }
    c.cBase.m_nBufPos = 0
    return c
}

func (p *NETInputPacket) ReadInt32() int32 {

    buf, err := p.cBase.read(4)
    if err != nil {
        fmt.Printf("%v\n", err)
        return 0
    }
    bufReader := bytes.NewReader(buf)
    var value int32 = 0
    err = binary.Read(bufReader, binary.BigEndian, &value)
    if err != nil {
        return 0
    }
    return value
}

func (p *NETInputPacket) ReadInt64() int64 {

    buf, err := p.cBase.read(8)
    if err != nil {
        return 0
    }
    bufReader := bytes.NewReader(buf)
    var value int64 = 0
    err = binary.Read(bufReader, binary.BigEndian, &value)
    if err != nil {
        return 0
    }
    return value
}

func (p *NETInputPacket) ReadIntShort() int16 {

    buf, err := p.cBase.read(2)
    if err != nil {
        return 0
    }
    bufReader := bytes.NewReader(buf)
    var value int16 = 0
    err = binary.Read(bufReader, binary.BigEndian, &value)
    if err != nil {
        return 0
    }
    return value
}

func (p *NETInputPacket) ReadByte() byte {

    buf, err := p.cBase.read(1)
    if err != nil {
        return byte(0)
    }
    return buf[0]
}

func (p *NETInputPacket) ReadBinary() ([]byte, error) {

    size := p.ReadInt32()
    buf, err := p.cBase.read(size)
    if err != nil {
        return nil, err
    }
    return buf, nil
}

func (p *NETInputPacket) Copy() ([]byte, error) {

    size := p.cBase.m_nHead.length
    buf, err := p.cBase.read(size)
    if err != nil {
        return nil, err
    }
    return buf, nil
}

func (p *NETInputPacket) GetcbCheckCode() byte {
    return p.cBase.m_nHead.code
}

func (p *NETInputPacket) GetCmd() int32 {
    return p.cBase.m_nHead.cmd
}

func (p *NETOutputPacket) EncryptBuffer() []byte {

    send := p.cBase.m_strBuf
    var code byte = 0
    sLen := len(send)
    pcbDataBuffer := make([]byte, sLen)
    var j int = 0
    for i := 0; i < sLen; i++ {
        code += send[i]
        pcbDataBuffer[j] = m_SendByteMap[int(send[i])]

        //fmt.Printf("i=%d, j=%d, send[i]=%x pcbDataBuffer[j]=%x\n", i, j, send[i], pcbDataBuffer[j])
        j = j + 1
    }
    code = (^code + 1)
    //fmt.Printf("code=%d . len[%d] [%x]\n", code, len(pcbDataBuffer),pcbDataBuffer)
    p.WritecbCheckCode(code)

    data := p.cBase.m_nHead.Serialize()
    data = append(data, pcbDataBuffer...)
    return data

}

func (p *NETInputPacket) DecryptBuffer() error {

    body := p.cBase.m_strBuf[0:]
    size := len(body)
    pcbDataBuffer := make([]byte, size)
    code := p.GetcbCheckCode()
    for i := 0; i < size; i++ {
        pcbDataBuffer[i] = m_RecvByteMap[body[i]]
        code += pcbDataBuffer[i]

    }
    if code == 0 {
        p.cBase.m_strBuf = pcbDataBuffer
        //fmt.Printf("%v\n",  p.cBase.m_strBuf)
        return nil
    } else {
        return errors.New("DecryptBuffer code error")
    }
}
