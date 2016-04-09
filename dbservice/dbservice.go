/**
 * 使用xorm数据库访问单元
 */
package dbservice

import (
	"fmt"

	_ "github.com/denisenkom/go-mssqldb" //mssql
	_ "github.com/go-sql-driver/mysql"   //mysql
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq" //postgres
	// _ "github.com/mattn/go-oci8"    // oracle，需安装pkg-config工具
	// _ "github.com/mattn/go-sqlite3" // sqlite
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

/**
 * 获取默认数据库引擎
 */
func (d *DBAccess) DefaultDB() *xorm.Engine {
	return d.Default
}

/**
 * 获取全部数据库引擎列表
 */
func (d *DBAccess) DBList() map[string]*xorm.Engine {
	return d.List
}

/**
 * 设置默认数据库引擎
 */
func (d *DBAccess) SetDefaultDB(name string) error {
	engine, ok := d.List[name]
	if !ok {
		return fmt.Errorf("Specified database does not exist: %v.", name)
	}
	d.Default = engine
	return nil
}

/**
 * 获取指定数据库引擎
 */
func (d *DBAccess) GetDB(name string) (*xorm.Engine, bool) {
	engine, ok := d.List[name]
	return engine, ok
}
