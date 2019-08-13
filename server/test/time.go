package main

import (
    "fmt"
    //"superserver/until/timingwheel"
    "time"
)

// func fff(a int) {
//     fmt.Println("------------------ a=%d", a)
// }

// func bb(a int) {
//     fmt.Println("------------------ a=", a)
// }

// type Server struct {
//     timerWheel *timingwheel.TimingWheel
// }

type GF interface{}

type Server struct {
    fn   GF
    args []interface{}
}

func change(args ...interface{}) {

    fmt.Println("----->", args)

}

func olld(a int, b string) {
    fmt.Println(a, b)
}

func main() {

    // tw := timingwheel.NewTimingWheel(time.Millisecond, 20)
    // tw.Start()
    // defer tw.Stop()

    // tw.AfterFunc(1*time.Second, bb, 10000)

    s := &Server{
        fn:   change,
        args: make([]interface{}, 0),
    }

    s.args = append(s.args, 100)
    s.args = append(s.args, "hhhhhhhhhhhhhhh")

    s.fn.(change)("1", "2", "3")

    //<-time.After(900 * time.Millisecond)
    // Stop the timer before it fires
    //t.Stop()

    for {
        time.Sleep(500 * time.Millisecond)
    }

}
