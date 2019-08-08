package main 
import (
  
  "time"
  "net"
  "fmt"
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
                fmt.Printf("Receive Signal: ", sig)
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
    fmt.Printf("Waiting For Singals\n")
    <-doneCh
    fmt.Printf("Need Exit\n")
}


func connoct() {
    conn, err := net.DialTimeout("tcp", "127.0.0.1:9100", 25*time.Second)
    if err != nil {
        fmt.Printf(" Connect failed: %v\n",err)
        return 
    }
    fmt.Printf("conn[%s]  success\n", conn.RemoteAddr().String())
    var buf = make([]byte,1024*64)
    for {
        n , err :=  conn.Read(buf)
        if n == 0 || err != nil {
            fmt.Printf("<%v>  err[%v]",conn.RemoteAddr(), err)
            return 
        } 

    }
}


func main() {


  for i := 0; i < 20000; i++ {
    go connoct() 
  }

   WaitSignalSynchronized()
}