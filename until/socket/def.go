package socket


import (
    "errors"
)

var (
    ErrParameter     = errors.New("parameter error")
    ErrNilKey        = errors.New("nil key")
    ErrNilValue      = errors.New("nil value")
    ErrWouldBlock    = errors.New("would block")
    ErrNotHashable   = errors.New("not hashable")
    ErrNilData       = errors.New("nil data")
    ErrBadData       = errors.New("more than 8M data")
    ErrNotRegistered = errors.New("handler not registered")
    ErrServerClosed  = errors.New("server has been closed")
)

const(
    HEAET_TIMER_OUT = iota
    RECONNECT_TIMER_OUT = 1

)


//OnConnectFunc 新链接建立成功回调
type OnConnectFunc func(int)

//OnCloseFunc 链接断开回调
type OnCloseFunc func(int)

//ConnWrapper 连接接口
type ConnWrapper interface {
    Start()
    Close()
    Write(*Message) error
    GetName() string
}
