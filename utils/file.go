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

package utils

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SelfPath gets compiled executable file absolute path
func SelfPath() string {
	path, _ := filepath.Abs(os.Args[0])
	return path
}

// SelfDir gets compiled executable file directory
func SelfDir() string {
	return filepath.Dir(SelfPath())
}

// 转相对路径
func RelPath(targpath string) string {
	basepath, _ := filepath.Abs("./")
	rel, _ := filepath.Rel(basepath, targpath)
	return strings.Replace(rel, `\`, `/`, -1)
}

//切换工作路径到自身所在目录下
var curpath = SelfDir()

func SelfChdir() {
	if err := os.Chdir(curpath); err != nil {
		log.Fatal(err)
	}
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// SearchFile Search a file in paths.
// this is often used in search config file in /etc ~/
func SearchFile(filename string, paths ...string) (fullpath string, err error) {
	for _, path := range paths {
		if fullpath = filepath.Join(path, filename); FileExists(fullpath) {
			return
		}
	}
	err = errors.New(fullpath + " not found in paths")
	return
}

// GrepFile like command grep -E
// for example: GrepFile(`^hello`, "hello.txt")
// \n is striped while read
func GrepFile(patten string, filename string) (lines []string, err error) {
	re, err := regexp.Compile(patten)
	if err != nil {
		return
	}

	fd, err := os.Open(filename)
	if err != nil {
		return
	}
	lines = make([]string, 0)
	reader := bufio.NewReader(fd)
	prefix := ""
	isLongLine := false
	for {
		byteLine, isPrefix, er := reader.ReadLine()
		if er != nil && er != io.EOF {
			return nil, er
		}
		if er == io.EOF {
			break
		}
		line := string(byteLine)
		if isPrefix {
			prefix += line
			continue
		} else {
			isLongLine = true
		}

		line = prefix + line
		if isLongLine {
			prefix = ""
		}
		if re.MatchString(line) {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// 遍历目录，可指定后缀，返回相对路径
func WalkDirs(targpath string, suffixes ...string) (dirlist []string) {
	if !filepath.IsAbs(targpath) {
		targpath, _ = filepath.Abs(targpath)
	}
	err := filepath.Walk(targpath, func(retpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return nil
		}
		if len(suffixes) == 0 {
			dirlist = append(dirlist, RelPath(retpath))
			return nil
		}
		_retpath := RelPath(retpath)
		for _, suffix := range suffixes {
			if strings.HasSuffix(_retpath, suffix) {
				dirlist = append(dirlist, _retpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("utils.WalkRelDirs: %v\n", err)
		return
	}

	return
}
