/**
 * 使用xorm数据库访问单元
 */
package lessgo

import (
	"database/sql"
	_ "github.com/go-xorm/core"
	_ "github.com/go-xorm/xorm"
)

/**
 * DBAccess 数据库访问管理
 */
type (
	DBAccess struct {
		DefaultDB *Engine
		ExtendDBs map[string]*Engine
	}
)

func DBAccess() *DBAccess {

}

/**
 * 根据数据库连接配置创建数据库连接
 */
func (this *DBAccess) InitDBAccess() {

}

/**
 * 根据 name获取DB访问实例
 */
func (this *DBAccess) ExtendDB(name string) *Engine {
	return this.ExtendDBs[name]
}

/**
 * 获取默认数据库访问实例
 */
func (this *DBAccess) DefaultDB() *Engine {
	return this.DefaultDB
}

/**
 * 释放
 */
func (this *DBAccess) Close() {

}
