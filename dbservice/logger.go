package dbservice

import (
	"fmt"

	"github.com/go-xorm/core"

	log "github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/logs/logs"
)

type ILogger struct {
	*logs.BeeLogger
	level   core.LogLevel
	showSQL bool
}

func NewILogger(l int) *ILogger {
	return &ILogger{
		BeeLogger: log.GlobalLogger.(*log.TgLogger).BeeLogger,
		level:     level(core.LogLevel(l)),
	}
}

func level(l core.LogLevel) core.LogLevel {
	switch int(l) {
	case log.DEBUG:
		return core.LOG_DEBUG
	case log.INFO:
		return core.LOG_INFO
	case log.WARN:
		return core.LOG_WARNING
	case log.ERROR:
		return core.LOG_ERR
	case log.OFF:
		return core.LOG_OFF
	}
	return core.LOG_UNKNOWN
}

func (i *ILogger) Debug(v ...interface{}) (err error) {
	i.BeeLogger.Debug(fmt.Sprintln(v...))
	return
}

func (i *ILogger) Debugf(format string, v ...interface{}) (err error) {
	i.BeeLogger.Debug(format, v...)
	return
}

func (i *ILogger) Err(v ...interface{}) (err error) {
	i.BeeLogger.Error(fmt.Sprintln(v...))
	return
}

func (i *ILogger) Errf(format string, v ...interface{}) (err error) {
	i.BeeLogger.Error(format, v...)
	return
}

func (i *ILogger) Info(v ...interface{}) (err error) {
	i.BeeLogger.Info(fmt.Sprintln(v...))
	return
}

func (i *ILogger) Infof(format string, v ...interface{}) (err error) {
	i.BeeLogger.Info(format, v...)
	return
}

func (i *ILogger) Warning(v ...interface{}) (err error) {
	i.BeeLogger.Warn(fmt.Sprintln(v...))
	return
}

func (i *ILogger) Warningf(format string, v ...interface{}) (err error) {
	i.BeeLogger.Warn(format, v...)
	return
}

func (i *ILogger) Level() core.LogLevel {
	return i.level
}

func (i *ILogger) SetLevel(l core.LogLevel) (err error) {
	i.level = level(l)
	i.BeeLogger.SetLevel(int(i.level))
	return
}

func (i *ILogger) ShowSQL(show ...bool) {
	if len(show) == 0 {
		i.showSQL = true
		return
	}
	i.showSQL = show[0]
}

func (i *ILogger) IsShowSQL() bool {
	return i.showSQL
}
