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

package logs

import (
	"encoding/json"

	"github.com/lessgo/lessgo/logs/color"
)

type (
	// // brush is a color join function
	brush func(msg interface{}, styles ...string) string
	// consoleWriter implements LoggerInterface and writes messages to terminal.
	consoleWriter struct {
		lg       *logWriter
		Level    int  `json:"level"`
		Colorful bool `json:"color"` //this filed is useful only when system's terminal supports color
	}
)

var (
	colors = []brush{
		color.White,   // LevelSystem        white
		color.RedBg,   // Fatal              redbg
		color.White,   // Emergency          white
		color.Cyan,    // Alert              cyan
		color.Magenta, // Critical           magenta
		color.Red,     // Error              red
		color.Yellow,  // Warning            yellow
		color.Blue,    // Notice             blue
		color.Green,   // Informational      green
		color.Green,   // Debug              green
	}
)
var out = newLogWriter(color.NewColorableStdout())

// NewConsole create ConsoleWriter returning as LoggerInterface.
func NewConsole() Logger {
	cw := &consoleWriter{
		lg:       out,
		Level:    LevelDebug,
		Colorful: true,
	}
	return cw
}

// Init init console logger.
// jsonConfig like '{"level":LevelTrace}'.
func (c *consoleWriter) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(jsonConfig), c)
	return err
}

// WriteMsg write message in console.
func (c *consoleWriter) WriteMsg(lm logMsg) error {
	if lm.level > c.Level {
		return nil
	}
	if c.Colorful {
		lm.line = color.Dim(lm.line)
		lm.prefix = colors[lm.level](lm.prefix)
	}
	c.lg.println(&lm)
	return nil
}

// Destroy implementing method. empty.
func (c *consoleWriter) Destroy() {

}

// Flush implementing method. empty.
func (c *consoleWriter) Flush() {

}

func init() {
	Register("console", NewConsole)
}
