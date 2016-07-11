Holmes
=======

Golang logging package


installation
------------
    go get github.com/leesper/holmes

Usage
-----
Add one line at the top of your main function, and you can do somg logging by calling such as holmes.Debug(...)

Features
-----
* Support creating new log file every hour/minute(rolling);
* Can also print to stdout while writing to file;
* Support levels: debug, info, warn, error, fatal;
* Can change log file path by passing LogFilePath("./log") to holmes.Start()
* Generating log files named PROGRAM.YYYY-MM-DD-HH-MM.PID.log

Things you can change
-----
It is by default, a debug-level and print-to-stdout logger, you can pass parameters to change its behavior:
* DebugLevel - change logger to debug level
* InfoLevel - change logger to info level
* WarnLevel - change logger to warn level
* ErrorLevel - change logger to error level
* FatalLevel - change logger to fatal level
* LogFilePath - make logger write to disk file
* EveryHour - logging to different file every hour
* EveryMinute - logging to different file every minute
* AlsoStdout - also logging to stdout


```go
import "github.com/leesper/holmes"

func main() {
  // log files put in ./log, create new one every hour, also print to stdout
  defer holmes.Start(LogFilePath("./log"), EveryHour, AlsoStdout).Stop()
  holmes.Info("%s", "If by life you were deceived,")
  holmes.Warn("%s", "Don't be dismal, don't be wild!")
  holmes.Error("%s", "In the day of grief, be mild.")
  holmes.Info("%s", "Merry days will come, believe.")
  ...
}
```
```
output:
2016/07/08 11:25:48  INFO [example.main] (example.go:48) - If by life you were deceived,
2016/07/08 11:25:48  WARN [example.main] (example.go:49) - Don't be dismal, don't be wild!
2016/07/08 11:25:48 ERROR [example.main] (example.go:50) - In the day of grief, be mild
2016/07/08 11:25:48  INFO [example.main] (example.go:51) - Merry days will come, believe.
```

Benchmark
-----
```
BenchmarkFileLoggerSingleGoroutine-4  	  100000	     17694 ns/op
BenchmarkFileLoggerMultipleGoroutine-4	   50000	     37284 ns/op
```
