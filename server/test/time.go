package main

import (
    "fmt"
    "superserver/until/timingwheel"
    "time"
)

func fff(a int) {
    fmt.Println("------------------ a=%d", a)
}

func bb(a int) {
    fmt.Println("------------------ a=", a)
}

type Server struct {
    timerWheel *timingwheel.TimingWheel
}

func main() {

    tw := timingwheel.NewTimingWheel(time.Millisecond, 20)
    tw.Start()
    defer tw.Stop()

    for i := 0; i < 100; i++ {
        tw.AfterFunc(1*time.Second, fff, i)
    }

    tw.AfterFunc(1*time.Second, bb, 10000)

    //<-time.After(900 * time.Millisecond)
    // Stop the timer before it fires
    //t.Stop()

    for {
        time.Sleep(500 * time.Millisecond)
    }

}
