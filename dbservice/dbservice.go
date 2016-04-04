/**
 * 使用xorm数据库访问单元
 */
package dbservice

import (
	"fmt"

	"github.com/go-xorm/xorm"

	"github.com/lessgo/lessgo"
)

/**
 * DBAccess 数据库访问管理
 */
type (
	DBAccess struct {
		Default *xorm.Engine
		List    map[string]*xorm.Engine
	}
)

var globalDBAccess = func() *DBAccess {
	access := &DBAccess{
		List: map[string]*xorm.Engine{},
	}
	for _, conf := range lessgo.AppConfig.DBList {
		engine, err := xorm.NewEngine(conf.Driver, conf.ConnString)
		if err != nil {
			lessgo.Logger().Error("%v", err)
			continue
		}
		engine.SetMaxOpenConns(conf.MaxOpenConns)
		engine.SetMaxIdleConns(conf.MaxIdleConns)
		access.List[conf.Name] = engine
		if lessgo.AppConfig.DefaultDB == conf.Name {
			access.Default = engine
		}
	}
	return access
}()

/**
 * 获取默认数据库引擎
 */
func DefaultDB() *xorm.Engine {
	return globalDBAccess.Default
}

/**
 * 获取全部数据库引擎列表
 */
func DBList() map[string]*xorm.Engine {
	return globalDBAccess.List
}

/**
 * 设置默认数据库引擎
 */
func SetDefaultDB(name string) error {
	engine, ok := globalDBAccess.List[name]
	if !ok {
		return fmt.Errorf("Specified database does not exist: %v.", name)
	}
	globalDBAccess.Default = engine
	return nil
}

/**
 * 获取指定数据库引擎
 */
func GetDB(name string) (*xorm.Engine, bool) {
	engine, ok := globalDBAccess.List[name]
	return engine, ok
}

// /**
//  * 根据数据库连接配置创建数据库连接
//  */
// func (this *DBAccess) InitDBAccess() {

// }

// /**
//  * 根据 name获取DB访问实例
//  */
// func (this *DBAccess) ExtendDB(name string) *Engine {
// 	return this.ExtendDBs[name]
// }

// /**
//  * 获取默认数据库访问实例
//  */
// func (this *DBAccess) DefaultDB() *Engine {
// 	return this.DefaultDB
// }

// /**
//  * 释放
//  */
// func (this *DBAccess) Close() {

// }
