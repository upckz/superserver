package common

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    log "superserver/until/czlog"
)

//CheckError if err is not nil, then log the detail and return true, otherwise return false
func CheckError(err error, desc string) (ret bool) {
    ret = false
    if err != nil {
        log.Errorf("Error[%s]: %s", desc, err.Error())
        ret = true
    }
    return
}

//CheckFileIsExist return true when filename exists, otherwise return false
func CheckFileIsExist(filename string) (exist bool) {
    exist = true
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        exist = false
    }
    return
}

func GetLogFileName(t time.Time) string {
    proc := filepath.Base(os.Args[0])
    now := time.Now()
    year := now.Year()
    month := now.Month()
    day := now.Day()
    hour := now.Hour()
    minute := now.Minute()
    pid := os.Getpid()
    return fmt.Sprintf("%s.%04d-%02d-%02d-%02d-%02d.%d",
        proc, year, month, day, hour, minute, pid)
}
