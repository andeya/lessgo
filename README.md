#Lessgo Web Framework  [![GoDoc](https://godoc.org/github.com/lessgo/lessgo?status.svg)](https://godoc.org/github.com/lessgo/lessgo)

![Lessgo Admin](https://github.com/lessgo/lessgo/raw/master/doc/favicon.png)

Lessgo 是一款 Go 语言编写的 web 快速开发框架。它基于开源框架 echo v2 进行二次开发，旨在实现一个简单、稳定、高效、灵活的 web 框架。在此感谢 [echo](https://github.com/labstack/echo)。它最显著的特点是支持Admin管理后台动态配置模块与操作的启用状态、中间件等。

* 官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

![Lessgo Admin](https://github.com/lessgo/lessgo/raw/master/doc/server.jpg)

![Lessgo Admin](https://github.com/lessgo/lessgo/raw/master/doc/admin.jpg)


##项目架构

![Lessgo Admin](https://github.com/lessgo/lessgo/raw/master/doc/ThinkgoWebFramework.jpg)



## 安装

1.下载框架源码
```sh
go get github.com/lessgo/lessgo
```

2.安装部署工具
```sh
go install
```

3.创建项目（在项目目录下运行cmd）
```sh
$ lessgo new appname
```

4.以热编译模式运行（在项目目录下运行cmd）
```sh
$ cd appname
$ lessgo run
```

##开源协议

Lessgo 项目采用商业应用友好的 [MIT](https://github.com/lessgo/lessgo/raw/master/doc/LICENSE) 协议发布。
