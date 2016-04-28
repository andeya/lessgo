# 数据库访问配置项(DB.config)
[defaultdb] --默认数据库
name=preset --数据库名称
tablesnake=true  --表名使用snake风格或保持不变
tablespace= --表命名空间
maxidleconns=1 --最大空闲连接数，超过该数量的会释放
showexectime=false --显示执行时间
showsql=false --显示执行的SQL
tablefix=prefix --表命名空间是前缀还是后缀：prefix | suffix
connstring=Common/DB/sqlite.db --连接字符串
disablecache=false --禁止缓存
columnpace=  -- 列命名空间
maxopenconns=1 --最大打开连接数
driver=sqlite3 --数据库驱动类型
columnsnake=true -- 列名使用snake风格或保持不变
columnfix=prefix 列命名空间是前缀还是后缀：prefix | suffix

