package lessgo

import (
	"errors"
	"github.com/lessgo/lessgo/logs"
)

func test1(ctx Context) error {
	logs.Info("路标：1")
	return nil
}

func test2(ctx Context) error {
	logs.Info("路标：2")
	return errors.New("text2 error")
}

func test3(ctx Context) error {
	logs.Info("路标：3")
	panic("panic test4")
	return nil
}

func test4(ctx Context) error {
	logs.Info("路标：4")
	return nil
}
