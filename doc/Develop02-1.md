# APP配置项(App.config)

[info] --框架基本信息
email=henrylee_cn@foxmail.com
license=MIT
licenseurl=https://github.com/lessgo/lessgo/raw/master/doc/LICENSE
version=0.4.0
description=A simple, stable, efficient and flexible web framework.
host=127.0.0.1:8080
contact=henrylee_cn@foxmail.com
termsofserviceurl=https://github.com/lessgo/lessgo

[filecache] --静态文件缓存配置项
maxcapmb=256 --最大缓存(内存MB)
cachesecond=600 --缓存的刷新时间(秒)
singlefileallowmb=64 --单文件缓存限制(单文件大小超过MB的不缓存)

[listen] --http服务器参数
enablehttps=false --允许https
httpscertfile=
httpskeyfile=
graceful=false
address=0.0.0.0:8080 --IP地址与端口
readtimeout=0 --读超时
writetimeout=0 --写超时

[session] --Session配置项
cookielifetime=3600
enablesetcookie=true
domain=
enable=false
cookiename=lessgosessionID
provider=memory --保存方式: memory=内存
providerconfig={"cookieName":"gosessionid", "enableSetCookie,omitempty": true, "gclifetime":3600, "maxLifetime": 3600, "secure": false, "sessionIDHashFunc": "sha1", "sessionIDHashKey": "", "cookieLifeTime": 3600, "providerConfig": ""}
gcmaxlifetime=3600

[log] --日志配置项
level=debug --日志级别
asyncchan=1000 --并发数开关

[system] --系统配置项
maxmemorymb=64  --最大内存
crossdomain=false --是否允许跨域调用
appname=lessgo   --应用名称
debug=true --调试模式
casesensitive=false  
