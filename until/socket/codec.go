package socket

import (
    "bytes"
    "encoding/binary"
    "errors"
    "github.com/xxtea/xxtea-go/xxtea"
)

func BytesCombine(pBytes ...[]byte) []byte {
    ln := len(pBytes)
    s := make([][]byte, ln)
    for index := 0; index < ln; index++ {
        s[index] = pBytes[index]
    }
    sep := []byte("")
    return bytes.Join(s, sep)
}

func MkMessage(msg *Message) ([]byte, error) {

    buffer := new(bytes.Buffer)
    err := binary.Write(buffer, binary.BigEndian, msg.cmd)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buffer, binary.BigEndian, msg.uid)
    if err != nil {
        return nil, err
    }

    err = binary.Write(buffer, binary.BigEndian, msg.data)
    if err != nil {
        return nil, err
    }
    return buffer.Bytes(), nil
}

// Encode from Message to []byte
func Encode(msg *Message) ([]byte, error) {

    pkg, err := MkMessage(msg)
    if err != nil {
        return nil, err
    }
    ln := len(pkg)

    buffer := new(bytes.Buffer)
    err = binary.Write(buffer, binary.BigEndian, int32(ln)+4) //pkg and head
    if err != nil {
        return nil, err
    }
    data := make([]byte, 0)
    data = append(data, buffer.Bytes()...)
    data = append(data, pkg...)

    return data, nil

}

// Decode from []byte to Message
func Decode(buff []byte) (*Message, error) {
    var err error
    size := len(buff)
    if size < 8 {
        size = 8
    }
    message := &Message{
        data: make([]byte, size-8),
    }

    bufReader := bytes.NewReader(buff)
    err = binary.Read(bufReader, binary.BigEndian, &message.cmd)
    if err != nil {
        return nil, err
    }
    err = binary.Read(bufReader, binary.BigEndian, &message.uid)
    if err != nil {
        return nil, err
    }

    err = binary.Read(bufReader, binary.BigEndian, &message.data)
    if err != nil {
        return nil, err
    }
    return message, nil
}

func EnBinaryPackage(msg *Message, opensslEncrypted []byte) ([]byte, error) {

    pkg, err := MkMessage(msg)
    if err != nil {
        return nil, err
    }

    dataBuffer := xxtea.Encrypt(pkg, opensslEncrypted)
    buffer := new(bytes.Buffer)
    err = binary.Write(buffer, binary.BigEndian, int32(len(dataBuffer)+4)) //pkg and head
    if err != nil {
        return nil, err
    }
    err = binary.Write(buffer, binary.BigEndian, dataBuffer)
    if err != nil {
        return nil, err
    }
    return buffer.Bytes(), nil
}

func DeBinaryPackage(msg, opensslEncrypted []byte) (*Message, error) {

    dataBuffer := xxtea.Decrypt(msg, opensslEncrypted)
    if len(dataBuffer) == 0 {
        return nil, errors.New("Decrypt error")
    }

    if len(dataBuffer) < 8 {
        return nil, errors.New("DeBinsryPackage size is error")
    }
    return Decode(dataBuffer)

}

func ParsePacket(data []byte) int32 {

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

func OnPacketComplete(data []byte, isSecert bool, secertKey []byte) (*Message, error) {
    if isSecert == true {
        return DeBinaryPackage(data[4:], secertKey)
    } else {
        return Decode(data[4:])
    }
}
