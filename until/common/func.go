package common

import (
   // "myserver/until/common"
   "time"
    "net/http"
    _ "net/http/pprof"
    log "superserver/until/czlog"
    "fmt"
)




const (
    SERVER_CENTER int32 = 1
    SERVER_AGENT int32 = 2
    SERVER_HALL  int32 = 3
    SERVER_ALLOC int32 = 4
    SERVER_GAME  int32 = 5 
    SERVER_USER  int32 = 6
    SERVER_MEMDATA int32 = 7
    SERVER_GOLD_UPDATE int32 = 8
    SERVER_LOG      int32 = 9
    SERVER_TJ       int32= 10
    SERVER_BROADCAST int32 = 11

)

const (
    TJ_USER_ONLINE int32 = 1
    TJ_USER_OFFLINE int32 = 2
    TJ_USER_PLAY    int32 = 3
    TJ_USER_LEAVE   int32 = 4
)


const (
    UPDATE_MONEY    int32 = 1
    UPDATE_SAFEBOX  int32 = 2
    UPDATE_EXP      int32 = 3
)

const (
    UPDATE_INT32    int32 = 1
    UPDATE_STRING   int32 = 2
)

func GetServerName(sverType int32) string{

    var name string =""
    switch(sverType) {
    case SERVER_CENTER:
        name = "CenterServer"
    case SERVER_AGENT:
        name = "AgentServer"
    case SERVER_HALL:
        name = "HallServer"
    case SERVER_ALLOC:
        name = "AllocServer"
    case SERVER_GAME:
        name = "GameServer"
    case SERVER_USER:
        name = "UserServer"
    case SERVER_MEMDATA:
        name = "MDataServer"
    case SERVER_LOG:
        name = "LogServer"
    case SERVER_TJ:
        name = "TJServer"
    case SERVER_BROADCAST:
        name = "BroadcastServer"
    case SERVER_GOLD_UPDATE:
        name = "GoldUpdateServer"

    }
    return name
}

  
func GetDelayTimeFrom24oClock() int64 {
    t := time.Now()
    tm1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
    tm2 := tm1.AddDate(0, 0, 1)
    return tm2.Unix()
}

func StartLogPerformance(logPath string, port int) bool {
    // profPath := fmt.Sprintf("%s/pprofLog/", logPath)
    // err := os.MkdirAll(profPath, os.ModePerm)
    // if err != nil {
    //  log.Errorln(err)
    //  return false
    // }
    // cpuFileName := fmt.Sprintf(profPath+"cpu_%s.prof", utils.GetLogFileName(time.Now()))
    // performance_cpu_file, err = os.OpenFile(cpuFileName, os.O_RDWR|os.O_CREATE, 0644)
    // if err != nil {
    //  log.Fatalln(err)
    //  return false
    // }
    // pprof.StartCPUProfile(performance_cpu_file)
    go func() {
        addr := fmt.Sprintf("0.0.0.0:%d", port)
        log.Infof("Will Start pprof on: %v", addr)
        err := http.ListenAndServe(addr, nil)
        if err != nil {
            log.Errorf("Start pprof failed. Error:%v", err)
        }
    }()
    return true
}
