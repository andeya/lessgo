// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logs provide a general log interface
// Usage:
//
// import "github.com/astaxie/beego/logs"
//
//	log := NewLogger(10000)
//	log.AddAdapter("console", "")
//
//	> the first params stand for how many channel
//
// Use it like this:
//
//	log.Trace("trace")
//	log.Info("info")
//	log.Warn("warning")
//	log.Debug("debug")
//	log.Critical("critical")
//
//  more docs http://beego.me/docs/module/logs.md
package logs

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// log message levels.
const (
	LevelSystem = iota
	LevelFatal
	LevelEmergency
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

var Prefix = map[int]string{
	LevelSystem:        "",
	LevelFatal:         "[F]",
	LevelEmergency:     "[M]",
	LevelAlert:         "[A]",
	LevelCritical:      "[C]",
	LevelError:         "[E]",
	LevelWarning:       "[W]",
	LevelNotice:        "[N]",
	LevelInformational: "[I]",
	LevelDebug:         "[D]",
}

type loggerType func() Logger

// Logger defines the behavior of a log provider.
type Logger interface {
	Init(config string) error
	WriteMsg(logMsg) error
	Destroy()
	Flush()
}

var adapters = make(map[string]loggerType)

// Register makes a log provide available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, log loggerType) {
	if log == nil {
		panic("logs: Register provide is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("logs: Register called twice for provider " + name)
	}
	adapters[name] = log
}

// BeeLogger is default logger in beego application.
// it can contain several providers and log message into all providers.
type BeeLogger struct {
	lock                sync.RWMutex
	level               int
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	msgChan             chan *logMsg
	signalChan          chan string
	wg                  sync.WaitGroup
	outputs             []*nameLogger
}

type nameLogger struct {
	Logger
	name string
}

type logMsg struct {
	level  int
	line   string
	prefix string
	msg    string
	when   time.Time
}

var logMsgPool = &sync.Pool{
	New: func() interface{} {
		return &logMsg{}
	},
}

// NewLogger returns a new BeeLogger.
// channelLen means the number of messages in chan(used where asynchronous is true).
// if the buffering chan is full, logger adapters write to file or other way.
func NewLogger(channelLen int64) *BeeLogger {
	bl := new(BeeLogger)
	bl.level = LevelDebug
	bl.loggerFuncCallDepth = 2
	bl.signalChan = make(chan string, 1)
	bl.msgChan = make(chan *logMsg, channelLen)
	bl.wg.Add(1)
	go bl.startLogger()
	return bl
}

func (bl *BeeLogger) SetMsgChan(channelLen int64) {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	bl.flush()
	bl.signalChan = make(chan string, 1)
	bl.msgChan = make(chan *logMsg, channelLen)
	bl.wg.Add(1)
	go bl.startLogger()
}

// AddAdapter provides a given logger adapter into BeeLogger with config string.
// config need to be correct JSON as string: {"interval":360}.
func (bl *BeeLogger) AddAdapter(adapterName string, config string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()

	for _, l := range bl.outputs {
		if l.name == adapterName {
			return fmt.Errorf("logs: duplicate adaptername %q (you have set this logger before)", adapterName)
		}
	}

	log, ok := adapters[adapterName]
	if !ok {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adapterName)
	}

	lg := log()
	err := lg.Init(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logs.BeeLogger.AddAdapter: "+err.Error())
		return err
	}
	bl.outputs = append(bl.outputs, &nameLogger{name: adapterName, Logger: lg})
	return nil
}

// DelLogger remove a logger adapter in BeeLogger.
func (bl *BeeLogger) DelAdapter(adapterName string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	outputs := []*nameLogger{}
	for _, lg := range bl.outputs {
		if lg.name == adapterName {
			lg.Destroy()
		} else {
			outputs = append(outputs, lg)
		}
	}
	if len(outputs) == len(bl.outputs) {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adapterName)
	}
	bl.outputs = outputs
	return nil
}

func (bl *BeeLogger) writeToLoggers(lm *logMsg) {
	for _, l := range bl.outputs {
		err := l.WriteMsg(*lm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to WriteMsg to adapter:%v,error:%v\n", l.name, err)
		}
	}
}

func (bl *BeeLogger) writeMsg(level int, msg string) {
	bl.lock.RLock()
	defer bl.lock.RUnlock()
	lm := logMsgPool.Get().(*logMsg)
	lm.when = time.Now()
	lm.level = level
	lm.prefix = Prefix[level]
	if bl.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(bl.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		lm.line = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]"
	}
	lm.msg = msg
	bl.msgChan <- lm
}

// SetLevel Set log message level.
// If message level (such as LevelDebug) is higher than logger level (such as LevelWarning),
// log providers will not even be sent the message.
func (bl *BeeLogger) SetLevel(l int) {
	bl.level = l
}

// SetLogFuncCallDepth set log funcCallDepth
func (bl *BeeLogger) SetLogFuncCallDepth(d int) {
	bl.loggerFuncCallDepth = d
}

// GetLogFuncCallDepth return log funcCallDepth for wrapper
func (bl *BeeLogger) GetLogFuncCallDepth() int {
	return bl.loggerFuncCallDepth
}

// EnableFuncCallDepth enable log funcCallDepth
func (bl *BeeLogger) EnableFuncCallDepth(b bool) {
	bl.enableFuncCallDepth = b
}

// start logger chan reading.
// when chan is not empty, write logs.
func (bl *BeeLogger) startLogger() {
	gameOver := false
	for {
		select {
		case bm := <-bl.msgChan:
			bl.writeToLoggers(bm)
			logMsgPool.Put(bm)
		case sg := <-bl.signalChan:
			// Now should only send "flush" or "close" to bl.signalChan
			bl.flush()
			if sg == "close" {
				for _, l := range bl.outputs {
					l.Destroy()
				}
				bl.outputs = nil
				gameOver = true
			}
			bl.wg.Done()
		}
		if gameOver {
			break
		}
	}
}

func (bl *BeeLogger) Sys(format string, v ...interface{}) {
	bl.writeMsg(LevelSystem, fmt.Sprintf(format, v...))
}

func (bl *BeeLogger) Fatal(format string, v ...interface{}) {
	if LevelFatal > bl.level {
		return
	}
	bl.writeMsg(LevelFatal, fmt.Sprintf(format, v...))
	bl.Flush()
	os.Exit(1)
}

// Emergency Log EMERGENCY level message.
func (bl *BeeLogger) Emergency(format string, v ...interface{}) {
	if LevelEmergency > bl.level {
		return
	}
	bl.writeMsg(LevelEmergency, fmt.Sprintf(format, v...))
}

// Alert Log ALERT level message.
func (bl *BeeLogger) Alert(format string, v ...interface{}) {
	if LevelAlert > bl.level {
		return
	}
	bl.writeMsg(LevelAlert, fmt.Sprintf(format, v...))
}

// Critical Log CRITICAL level message.
func (bl *BeeLogger) Critical(format string, v ...interface{}) {
	if LevelCritical > bl.level {
		return
	}
	bl.writeMsg(LevelCritical, fmt.Sprintf(format, v...))
}

// Error Log ERROR level message.
func (bl *BeeLogger) Error(format string, v ...interface{}) {
	if LevelError > bl.level {
		return
	}
	bl.writeMsg(LevelError, fmt.Sprintf(format, v...))
}

// Warn Log WARN level message.
// compatibility alias for Warning()
func (bl *BeeLogger) Warn(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	bl.writeMsg(LevelWarning, fmt.Sprintf(format, v...))
}

// Notice Log NOTICE level message.
func (bl *BeeLogger) Notice(format string, v ...interface{}) {
	if LevelNotice > bl.level {
		return
	}
	bl.writeMsg(LevelNotice, fmt.Sprintf(format, v...))
}

// Info Log INFO level message.
// compatibility alias for Informational()
func (bl *BeeLogger) Info(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	bl.writeMsg(LevelInformational, fmt.Sprintf(format, v...))
}

// Debug Log DEBUG level message.
func (bl *BeeLogger) Debug(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	bl.writeMsg(LevelDebug, fmt.Sprintf(format, v...))
}

// 简单实现io.Writer接口
func (bl *BeeLogger) Write(p []byte) (n int, err error) {
	bl.writeMsg(LevelSystem, string(p))
	return len(p), nil
}

// Flush flush all chan data.
func (bl *BeeLogger) Flush() {
	bl.signalChan <- "flush"
	bl.wg.Wait()
	bl.wg.Add(1)
}

// Close close logger, flush all chan data and destroy all adapters in BeeLogger.
func (bl *BeeLogger) Close() {
	bl.signalChan <- "close"
	bl.wg.Wait()
	close(bl.msgChan)
	close(bl.signalChan)
}

// Reset close all outputs, and set bl.outputs to nil
func (bl *BeeLogger) Reset() {
	bl.Flush()
	for _, l := range bl.outputs {
		l.Destroy()
	}
	bl.outputs = nil
}

func (bl *BeeLogger) flush() {
	for {
		if len(bl.msgChan) > 0 {
			bm := <-bl.msgChan
			bl.writeToLoggers(bm)
			logMsgPool.Put(bm)
			continue
		}
		break
	}
	for _, l := range bl.outputs {
		l.Flush()
	}
}
