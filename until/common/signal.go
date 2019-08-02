package common

import (
    log "superserver/until/czlog"
    "os"
    "os/signal"
    "syscall"
)

//WaitSignalSynchronized 同步等待系统信号并处理
func WaitSignalSynchronized() {
    sigsCh := make(chan os.Signal, 1)
    doneCh := make(chan struct{})

    signal.Notify(sigsCh, syscall.SIGABRT, syscall.SIGPIPE, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.Signal(10))

    go func() {
        for {
            select {
            // case <-time.After(30 * time.Second):
            //  log.Infoln("Time Out Close")
            //  close(doneCh)
            //  return
            case sig := <-sigsCh:
                log.Warnln("Receive Signal: ", sig)
                switch sig {
                case syscall.Signal(10):
                    close(doneCh)
                    return
                case syscall.SIGINT:
                    close(doneCh)
                    return
                default:
                    continue
                }
            }
        }
    }()
    log.Warnln("Waiting For Singals")
    <-doneCh
    log.Warnln("Need Exit")
}
