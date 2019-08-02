package main 
import (

    "time"
    "context"
    "log"
    "os"
)

var logg *log.Logger

type Server struct {

    ctx           context.Context
    cancel        context.CancelFunc
}

func(s *Server)startEpoll(i int) {
  log.Printf("startEpoll start...")
  defer func() {
        log.Printf("exit[%d] startEpoll", i)
        s.Close()
    }()
    go s.Start()

    for {
       select {
        case <-s.ctx.Done():
            return 
        default:
        }
    }

}

func (s *Server)Start() {
     defer func() {
        log.Printf("exit start")
        s.Close()
    }()


   for {
        select {
        case <-s.ctx.Done():
            return 
        default:
        }
       

    }
}

func (s *Server)Close() {
  s.cancel()
  log.Printf("Server Has Close")
}

func (s *Server)Run() {
  for i := 0; i < 2; i++ {
    go s.startEpoll()
  }

}

func main() {

  logg = log.New(os.Stdout, "", log.Ltime)
  ctx, cancel := context.WithCancel(context.Background())
  

  s := &Server{}
  s.ctx, s.cancel = context.WithCancel(ctx)
  s.Run()
  time.Sleep(2*time.Second)


  cancel()
  logg.Printf("down")
  for {

  }

}