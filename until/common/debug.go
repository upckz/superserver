package common

import (
    "bytes"
    "fmt"
    "runtime"
)

//GlobalPanicLog 打印错误日志
func GlobalPanicLog(p interface{}) string {
    return fmt.Sprintf("%s\n%s", p, string(PanicTrace(2)))
}

// PanicTrace trace panic stack info.
func PanicTrace(kb int) []byte {
    s := []byte("/src/runtime/panic.go")
    e := []byte("\ngoroutine ")
    line := []byte("\n")
    stack := make([]byte, kb<<10) //4KB
    length := runtime.Stack(stack, true)
    start := bytes.Index(stack, s)
    stack = stack[start:length]
    start = bytes.Index(stack, line) + 1
    stack = stack[start:]
    end := bytes.LastIndex(stack, line)
    if end != -1 {
        stack = stack[:end]
    }
    end = bytes.Index(stack, e)
    if end != -1 {
        stack = stack[:end]
    }
    stack = bytes.TrimRight(stack, "\n")
    return stack
}
