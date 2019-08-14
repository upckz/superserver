package main

import (
    "fmt"
    "superserver/until/socket"
)

func main() {

    s := socket.NewServerManger()

    s.InsertServer(1, 200, 201, 1, 10)

    fmt.Println(s.GetConnId(1, 201, 201, 1))

    // fmt.Println(s.FindServer(1))
    // fmt.Println(s.FindServer(1).FindServer(200))
    // fmt.Println(s.FindServer(1).FindServer(200).FindServer(201))
    // s.Remove(1, 200, 201, 1)
    // fmt.Println(s.FindServer(1).FindServer(201).FindServer(201).GetConnId(1))

}
