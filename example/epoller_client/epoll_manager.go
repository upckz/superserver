package epoller_client

import (
    "context"
    "errors"
    "fmt"
    "github.com/orcaman/concurrent-map"
    "superserver/until/common"
    log "superserver/until/czlog"
    "superserver/until/epoll"
)

type CliMsgWrapper struct {
    ID      int
    Msg     *common.Message
    SendAll bool
}

type ConfigOfClient struct {
    Ip            string //连接的ip
    Port          int32  //连接的port
    ServerType    int32  //连接server类型
    Svid          int32  //游戏的svid
    Gameid        int32  //游戏id
    Level         int32  //场次等级
    Secert        bool   //通讯是否加密
    ReconnectFlag bool   //是否重连
    HbTimeout     int32  //heaer time 心跳时间间隔
}

type Instance struct {
    clientMap         cmap.ConcurrentMap  //管理client map
    clientOnConnectCh chan int            //同时建立连接数
    clientReadCh      chan *CliMsgWrapper //读chan 大小
    clientWriteCh     chan *CliMsgWrapper //写chan 大小
    netIdentifier     *common.AtomicUint64
}

//newInstance
func NewClientInstance(ctx context.Context, dataSize int32) *Instance {
    if dataSize <= 0 {
        dataSize = 10
    }
    i := &Instance{
        clientMap:         cmap.New(),
        clientOnConnectCh: make(chan int, 100),
        clientReadCh:      make(chan *CliMsgWrapper, dataSize),
        clientWriteCh:     make(chan *CliMsgWrapper, dataSize),
        netIdentifier:     common.NewAtomicUint64(1),
    }

    return i
}

//add client
func (i *Instance) AddClientWith(cfg *ConfigOfClient, epoller *epoll.Epoll, nn int32) error {

    idx := i.netIdentifier.GetAndIncrement() + uint64(nn)
    cli := NewClient(i, cfg, epoller, idx)
    if cli != nil {
        return i.AddMapCli(cli)
    }
    return errors.New("add client error")
}

func (i *Instance) LenMapCli() int32 {
    return int32(i.clientMap.Count())
}

func (i *Instance) AddMapCli(cli *Client) error {
    if cli != nil {
        fd := cli.GetFD()
        if err := cli.Epoller.Add(fd); err != nil {
            cli.Close()
            return err
        }

        key := fmt.Sprint(fd)
        i.clientMap.Set(key, cli)
    }
    return nil
}

func (i *Instance) RemoveMapCli(cli *Client) error {
    if cli != nil {
        fd := cli.GetFD()
        if err := cli.Epoller.Remove(fd); err != nil {
            return err
        }
        key := fmt.Sprint(fd)
        i.clientMap.Remove(key)
    }
    return nil
}

func (i *Instance) OnCloseCli(cli *Client) {
    if cli != nil {
        i.RemoveMapCli(cli)
    }
}
func (i *Instance) NoticeToConnect(cli *Client) {
    if cli != nil {
        i.clientOnConnectCh <- cli.GetFD()
    }
}

func (i *Instance) GetMapCli(fd int) *Client {
    key := fmt.Sprint(fd)
    cli, ok := i.clientMap.Get(key)
    if ok {
        return cli.(*Client)
    }
    return nil
}

func (i *Instance) FindMapCli(serverType int32, gameid int32, svid int32, level int32) *Client {

    for item := range i.clientMap.IterBuffered() {
        val := item.Val
        if val == nil {
            continue
        }
        tmp := val.(*Client)
        if tmp != nil && tmp.svid == svid && tmp.svrType == serverType && tmp.gameid == gameid && tmp.level == level {
            return tmp
        }
    }
    return nil
}

func (i *Instance) GetCliByGameIdAndLevel(svrType int32, gameid int32) []*Client {

    var connMap []*Client
    for item := range i.clientMap.IterBuffered() {
        val := item.Val
        if val == nil {
            continue
        }
        tmp := val.(*Client)
        if tmp != nil && tmp.svrType == svrType && tmp.gameid == gameid {
            connMap = append(connMap, tmp)
        }
    }
    return connMap
}

func (i *Instance) GetCliBySvrType(svrType int32) []*Client {

    var connMap []*Client
    for item := range i.clientMap.IterBuffered() {
        val := item.Val
        if val == nil {
            continue
        }
        tmp := val.(*Client)
        if tmp == nil || tmp.GetSvrType() != svrType {
            continue
        }
        connMap = append(connMap, tmp)
    }
    return connMap
}

func (i *Instance) GetAllCli() []*Client {

    var connMap []*Client
    for item := range i.clientMap.IterBuffered() {
        val := item.Val
        if val == nil {
            continue
        }
        tmp := val.(*Client)
        if tmp == nil {
            continue
        }
        connMap = append(connMap, tmp)
    }
    return connMap
}

//GetCliName 获得Cli名字
func (i *Instance) GetCliName(idx int) string {
    key := fmt.Sprint(idx)
    c, ok := i.clientMap.Get(key)
    if ok {
        cli := c.(*Client)
        return cli.GetName()
    }
    return ""
}

//onconnect cli
func (i *Instance) OnConnect() <-chan int {
    return i.clientOnConnectCh
}

//on read pack
func (i *Instance) OnReadPacket() <-chan *CliMsgWrapper {
    return i.clientReadCh
}

//write pkg
func (i *Instance) OnWritePacket(msg *CliMsgWrapper) {
    log.Debugf("------ readchan len[%d] [%v]", len(i.clientReadCh), msg)
    i.clientReadCh <- msg
}

//send pack
func (i *Instance) WritePacket(pkt *CliMsgWrapper) (bool, int) {
    var cli *Client

    if pkt.SendAll {
        var count int
        for item := range i.clientMap.IterBuffered() {
            val := item.Val
            if val == nil {
                continue
            }
            cli := val.(*Client)
            if cli.HasConnect() {
                count++
                cli.SendMessage(pkt.Msg)
            }
        }
        return true, count

    } else {
        if pkt.ID != 0 {
            cli = i.GetMapCli(pkt.ID)
        }
        if cli != nil {
            cli.SendMessage(pkt.Msg)
            return true, cli.GetFD()
        }
    }
    return false, 0
}

//close
func (i *Instance) Close() {
    for item := range i.clientMap.IterBuffered() {
        val := item.Val
        if val == nil {
            continue
        }
        cli := val.(*Client)
        i.RemoveMapCli(cli)

    }
    close(i.clientOnConnectCh)
    i.clientOnConnectCh = nil
    close(i.clientReadCh)
    i.clientReadCh = nil
    close(i.clientWriteCh)
    i.clientWriteCh = nil
}