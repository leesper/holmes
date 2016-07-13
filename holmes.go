package holmes

import (
  "fmt"
  "log"
  "os"
  "path"
  "runtime"
  "sync/atomic"
  "time"
)

const (
  DEBUG LogLevel = iota
  INFO
  WARN
  ERROR
  FATAL
)

var (
  started int32
  loggerInstance innerLogger
  tagName = map[LogLevel]string{
    DEBUG: "DEBUG",
    INFO: "INFO",
    WARN: "WARN",
    ERROR: "ERROR",
    FATAL: "FATAL",
  }
)

type LogLevel int

func Start(decorators ...func(innerLogger) innerLogger) innerLogger {
  if atomic.CompareAndSwapInt32(&started, 0, 1) {
    loggerInstance = innerLogger{}
    for _, decorator := range(decorators) {
      loggerInstance = decorator(loggerInstance)
    }
    var logger *log.Logger
    var segment *logSegment
    if loggerInstance.logPath != "" {
      segment = newLogSegment(loggerInstance.unit, loggerInstance.logPath)
    }
    if segment != nil {
      logger = log.New(segment, "", log.LstdFlags)
    } else {
      logger = log.New(os.Stderr, "", log.LstdFlags)
    }
    loggerInstance.logger = logger
    return loggerInstance
  }
  panic("Start() already called")
}

func (l innerLogger)Stop() {
  if atomic.CompareAndSwapInt32(&l.stopped, 0, 1) {
    if l.printStack {
      traceInfo := make([]byte, 1 << 16)
      n := runtime.Stack(traceInfo, true)
      l.logger.Printf("%s", traceInfo[:n])
      if l.isStdout {
        log.Printf("%s", traceInfo[:n])
      }
    }
    if l.segment != nil {
      l.segment.Close()
    }
    l.segment = nil
    l.logger = nil
    atomic.StoreInt32(&started, 0)
  }
}

// io.Writer
type logSegment struct{
  unit time.Duration
  logPath string
  logFile *os.File
  timeToCreate <-chan time.Time
}

func newLogSegment(unit time.Duration, logPath string) *logSegment {
  now := time.Now()
  if logPath != "" {
    err := os.MkdirAll(logPath, os.ModePerm)
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      return nil
    }
    name := getLogFileName(time.Now())
    logFile, err := os.OpenFile(path.Join(logPath, name), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
      if os.IsNotExist(err) {
        logFile, err = os.Create(path.Join(logPath, name))
        if err != nil {
          fmt.Fprintln(os.Stderr, err)
          return nil
        }
      } else {
        fmt.Fprintln(os.Stderr, err)
        return nil
      }
    }
    next := now.Truncate(unit).Add(unit)
    var timeToCreate <-chan time.Time
    if unit == time.Hour || unit == time.Minute {
      timeToCreate = time.After(next.Sub(time.Now()))
    }
    return &logSegment{
      unit: unit,
      logPath: logPath,
      logFile: logFile,
      timeToCreate: timeToCreate,
    }
  }
  return nil
}

func (ls *logSegment)Write(p []byte) (n int, err error) {
  if ls.timeToCreate != nil && ls.logFile != os.Stdout && ls.logFile != os.Stderr {
    select {
    case current := <-ls.timeToCreate:
      ls.logFile.Close()
      ls.logFile = nil
      name := getLogFileName(current)
      ls.logFile, err = os.Create(path.Join(ls.logPath, name))
      if err != nil { // log into stderr if we can't create new file
        fmt.Fprintln(os.Stderr, err)
        ls.logFile = os.Stderr
      } else {
        next := current.Truncate(ls.unit).Add(ls.unit)
        ls.timeToCreate = time.After(next.Sub(time.Now()))
      }
    default:
      // do nothing
    }
  }
  return ls.logFile.Write(p)
}

func (fs *logSegment)Close() {
  fs.logFile.Close()
}

func getLogFileName(t time.Time) string {
  proc := path.Base(os.Args[0])
  now := time.Now()
  year := now.Year()
  month := now.Month()
  day := now.Day()
  hour := now.Hour()
  minute := now.Minute()
  pid := os.Getpid()
  return fmt.Sprintf("%s.%04d-%02d-%02d-%02d-%02d.%d.log",
    proc, year, month, day, hour, minute, pid)
}

type innerLogger struct{
  logger *log.Logger
  level LogLevel
  segment *logSegment
  stopped int32
  logPath string
  unit time.Duration
  isStdout bool
  printStack bool
}

func (l innerLogger)doPrintf(level LogLevel, format string, v ...interface{}) {
  if l.logger == nil {
    return
  }
  if level >= l.level {
    funcName, fileName, lineNum := getRuntimeInfo()
    format = fmt.Sprintf("%5s [%s] (%s:%d) - %s", tagName[level], path.Base(funcName), path.Base(fileName), lineNum, format)
    l.logger.Printf(format, v...)
    if l.isStdout {
      log.Printf(format, v...)
    }
    if level == FATAL {
      os.Exit(1)
    }
  }
}

func (l innerLogger)doPrintln(level LogLevel, v ...interface{}) {
  if l.logger == nil {
    return
  }
  if level >= l.level {
    funcName, fileName, lineNum := getRuntimeInfo()
    prefix := fmt.Sprintf("%5s [%s] (%s:%d) - ", tagName[level], path.Base(funcName), path.Base(fileName), lineNum)
    value := fmt.Sprintf("%s %s", prefix, fmt.Sprint(v...))
    l.logger.Println(value)
    if l.isStdout {
      log.Println(value)
    }
    if level == FATAL {
      os.Exit(1)
    }
  }
}

func getRuntimeInfo() (string, string, int) {
  pc, fn, ln, ok := runtime.Caller(3)  // 3 steps up the stack frame
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

func DebugLevel(l innerLogger) innerLogger {
  l.level = DEBUG
  return l
}

func InfoLevel(l innerLogger) innerLogger {
  l.level = INFO
  return l
}

func WarnLevel(l innerLogger) innerLogger {
  l.level = WARN
  return l
}

func ErrorLevel(l innerLogger) innerLogger {
  l.level = ERROR
  return l
}

func FatalLevel(l innerLogger) innerLogger {
  l.level = FATAL
  return l
}

func LogFilePath(p string) func(innerLogger) innerLogger {
  return func(l innerLogger) innerLogger {
    l.logPath = p
    return l
  }
}

func EveryHour(l innerLogger) innerLogger {
  l.unit = time.Hour
  return l
}

func EveryMinute(l innerLogger) innerLogger {
  l.unit = time.Minute
  return l
}

func AlsoStdout(l innerLogger) innerLogger {
  l.isStdout = true
  return l
}

func PrintStack(l innerLogger) innerLogger {
  l.printStack = true
  return l
}

func Debug(format string, v ...interface{}) {
  loggerInstance.doPrintf(DEBUG, format, v...)
}

func Info(format string, v ...interface{}) {
  loggerInstance.doPrintf(INFO, format, v...)
}

func Warn(format string, v ...interface{}) {
  loggerInstance.doPrintf(WARN, format, v...)
}

func Error(format string, v ...interface{}) {
  loggerInstance.doPrintf(ERROR, format, v...)
}

func Fatal(format string, v ...interface{}) {
  loggerInstance.doPrintf(FATAL, format, v...)
}

func Debugln(v ...interface{}) {
  loggerInstance.doPrintln(DEBUG, v...)
}

func Infoln(v ...interface{}) {
  loggerInstance.doPrintln(INFO, v...)
}

func Warnln(v ...interface{}) {
  loggerInstance.doPrintln(WARN, v...)
}

func Errorln(v ...interface{}) {
  loggerInstance.doPrintln(ERROR, v...)
}

func Fatalln(v ...interface{}) {
  loggerInstance.doPrintln(FATAL, v...)
}
