package logs

import (
	"github.com/henrylee2cn/lessgo/logs/logs"
)

type (
	Logger interface {
		SetMsgChan(channelLen int64)
		// SetLevel Set log message level.
		// If message level (such as LevelDebug) is higher than logger level (such as LevelWarning),
		// log providers will not even be sent the message.
		SetLevel(l int)
		// EnableFuncCallDepth enable log funcCallDepth
		EnableFuncCallDepth(b bool)
		// AddAdapter provides a given logger adapter into Logger with config string.
		// config need to be correct JSON as string: {"interval":360}.
		AddAdapter(adaptername string, config string) error

		Write(p []byte) (n int, err error)
		Sys(format string, v ...interface{})
		Fatal(format string, v ...interface{})
		Error(format string, v ...interface{})
		Warn(format string, v ...interface{})
		Info(format string, v ...interface{})
		Debug(format string, v ...interface{})
	}

	TgLogger struct {
		*logs.BeeLogger
	}
)

// Log levels to control the logging output.
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

func NewLogger(channelLen int64) Logger {
	tl := &TgLogger{logs.NewLogger(channelLen)}
	tl.BeeLogger.SetLogFuncCallDepth(3)
	return tl
}

func (t *TgLogger) SetLevel(l int) {
	t.BeeLogger.SetLevel(ExchangeLevel(l))
}

func ExchangeLevel(l int) int {
	switch l {
	case DEBUG:
		return logs.LevelDebug
	case INFO:
		return logs.LevelInformational
	case WARN:
		return logs.LevelWarning
	case ERROR:
		return logs.LevelError
	case FATAL:
		return logs.LevelFatal
	case OFF:
		return logs.LevelEmergency - 1
	}
	return logs.LevelError
}
