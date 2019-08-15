package log

import (
    "fmt"
    "github.com/sirupsen/logrus"
    "os"
    "path"
    "path/filepath"
    "runtime"
    "strconv"
    "time"
)

type logFile struct {
    logfile  *os.File
    fileName string
    path     string
    Log      *logrus.Logger
}
type Fields = logrus.Fields

var logOb *logFile

func init() {
    logOb = &logFile{
        Log: logrus.New(),
    }
    var environment string
    environment = os.Getenv("Development")
    if environment == "Development" {
        logOb.Log.Formatter = &logrus.TextFormatter{
            DisableColors: true,
            FullTimestamp: true,
        }
    } else {
        logOb.Log.Formatter = &logrus.JSONFormatter{
            TimestampFormat: "2006-01-02 15:04:05.9999",
        }
    }
    logOb.Log.SetLevel(logrus.DebugLevel)
}

func SetPath(path, filename string, expiredDay int) {
    logOb.path = path
    logOb.fileName = filename
    openFile()
    if expiredDay <= 0 {
        expiredDay = 7
    }
    go setLogExpiredDay(expiredDay)
}

func SetLevel(level logrus.Level) {
    logOb.Log.SetLevel(level)
}

func setLogExpiredDay(day int) {
    for {
        path := logOb.path + time.Now().AddDate(0, 0, -day).Format("20060102") + "/"
        filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
            if nil == fi {
                return err
            }
            if !fi.IsDir() {
                return nil
            }
            os.RemoveAll(path)
            return nil
        })
        <-time.After(24 * time.Hour)
    }
}

func openFile() {
    pid := strconv.Itoa(os.Getpid())
    dir := logOb.path + time.Now().Format("20060102") + "/"
    exist, _ := pathExists(dir)
    if exist == false {
        os.MkdirAll(dir, os.ModePerm)
    }
    logfile, _ := os.OpenFile(dir+logOb.fileName+"_"+time.Now().Format("150405")+"_"+pid, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
    logOb.Log.Out = logfile
    logOb.logfile.Close()
    logOb.logfile = logfile
    go dayRotatelogs()
}

func pathExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, err
}

func dayRotatelogs() {
    timeStr := time.Now().Format("2006-01-02")
    todayTime, _ := time.ParseInLocation("2006-01-02", timeStr, time.Local)
    interval := todayTime.AddDate(0, 0, 1).Unix() - time.Now().Unix()
    <-time.After(time.Duration(interval) * time.Second)
    openFile()
}

func Debug(v interface{}) {
    logOb.Log.Debug(v)
}
func Info(v interface{}) {
    logOb.Log.Info(v)
}
func Warn(v interface{}) {
    logOb.Log.Warn(v)
}
func Error(v interface{}) {
    logOb.Log.Error(v)
}
func Fatal(v interface{}) {
    logOb.Log.Fatal(v)
}
func Panic(v interface{}) {
    logOb.Log.Panic(v)
}
func WithFields(fields Fields) *logrus.Entry {
    funcName, _, lineNum := getRuntimeInfo()
    prefix := fmt.Sprintf("[%s:%d]", path.Base(funcName), lineNum)
    fields["_func"] = prefix
    return logOb.Log.WithFields(fields)
}

func getRuntimeInfo() (string, string, int) {
    pc, fn, ln, ok := runtime.Caller(2) // 3 steps up the stack frame
    if !ok {
        fn = "???"
        ln = 0
    }
    function := "???"
    caller := runtime.FuncForPC(pc)
    if caller != nil {
        function = caller.Name()
    }
    return function, fn, ln
}
