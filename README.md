#Lessgo Web Framework  [![GoDoc](https://godoc.org/github.com/lessgo/lessgo?status.svg)](https://godoc.org/github.com/lessgo/lessgo)[![GitHub release](https://img.shields.io/github/release/lessgo/lessgo.svg)](https://github.com/lessgo/lessgo/releases)

![Lessgo Favicon](https://github.com/lessgo/lessgo/raw/master/doc/favicon.png)

Lessgo 是 Go 语言编写的一款简单、稳定、高效、灵活的 web 完全开发框架。它博采众长，核心架构改写自[echo v2](https://github.com/labstack/echo)，数据库内置为[xorm](https://github.com/go-xorm/xorm)，模板引擎内置为[pongo2](https://github.com/flosch/pongo2)，其他某些功能模块改写自[beego](https://github.com/astaxie/beego)以及其他优秀开源项目。

Lessgo推荐使用函数式编程风格进行Web应用开发，HandlerFunc与MiddlewareFunc是一个项目的基础组成单元。它在项目组织上完美兼容网站全站与仅API两种类型的项目部署，同时支持运行时动态重建路由，这意味着后期运维可以在Admin中进行配置中间件、禁用与启用模块/操作等！（Lessgo在此感谢那些优秀开源项目的支持）

* 官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

![Lessgo Server](https://github.com/lessgo/lessgo/raw/master/doc/server.jpg)



## 安装

1.下载框架源码
```sh
go get github.com/lessgo/lessgo
go get github.com/lessgo/lessgoext/...
```

2.安装部署工具
```sh
cd %GOPATH%/github.com/lessgo/lessgoext/lessgo
go install
```
(该工具将会自动创建一套Demo，以供学习与开发)

3.创建项目（在项目目录下运行cmd）
```sh
$ lessgo new appname
```

4.以热编译模式运行（在项目目录下运行cmd）
```sh
$ cd appname
$ lessgo run
```

##项目组织目录

```
─Project 项目开发目录
├─Config 配置文件目录
│  ├─app.config 系统应用配置文件
│  └─db.config 数据库配置文件
├─Common 后端公共目录
│  ├─Middleware 中间件目录
│  └─... 其他
├─Static 前端公共目录 (url: /static)
│  ├─Tpl 公共tpl模板目录
│  ├─Js 公共js目录 (url: /static/js)
│  ├─Css 公共css目录 (url: /static/css)
│  ├─Img 公共img目录 (url: /static/img)
│  └─Plugin 公共js插件 (url: /static/plugin)
├─SystemAPI 系统模块后端目录
│  ├─SysRouter.go 系统模块路由文件
│  ├─Xxx Xxx子模块目录
│  │  ├─ExampleHandle.go Example操作
│  │  ├─ExampleModel.go Example数据模型及模板函数
│  │  └─... Xxx的子模块目录
│  └─... 其他子模块目录
├─SystemView 系统模块前端目录 (url: /system)
│  ├─Xxx Xxx子模块目录 (url: /system/xxx)
│  │  ├─example.tpl ExampleHandle对应的模板文件
│  │  ├─example2.html 无需绑定操作的静态html文件
│  │  ├─xxx.css css文件(可有多个)
│  │  ├─xxx.js js文件(可有多个)
│  │  └─... Xxx的子模块目录
├─BusinessAPI 业务模块后端目录
│  ├─BusRouter.go 业务模块路由文件
│  ├─Xxx Xxx子模块目录
│  │  ├─ExampleHandle.go Example操作
│  │  ├─ExampleModel.go Example数据模型及模板函数
│  │  └─... Xxx的子模块目录
│  └─... 其他子模块目录
├─BusinessView 业务模块前端目录 (url: /business)
│  ├─Xxx Xxx子模块目录 (url: /business/xxx)
│  │  ├─example.tpl ExampleHandle对应的模板文件
│  │  ├─example2.html 无需绑定操作的静态html文件
│  │  ├─xxx.css css文件(可有多个)
│  │  ├─xxx.js js文件(可有多个)
│  │  └─... Xxx的子模块目录
├─Uploads 默认上传下载目录
├─Logger 运行日志输出目录
└─Main.go 应用入口文件
```

##框架相关

* 核心框架：[lessgo](https://github.com/lessgo/lessgo)
* 框架扩展：[lessgoext](https://github.com/lessgo/lessgoext)
* 项目Demo：[demo](https://github.com/lessgo/lessgoext)


##项目架构

![Lessgo Web Framework](https://github.com/lessgo/lessgo/raw/master/doc/LessgoWebFramework.jpg)



#### 贡献者名单

贡献者                          |贡献概要
--------------------------------|--------------------------------------------------
henrylee2cn|第一作者 (主要的代码实现者) 
changyu72|第二作者 (主要的架构设计者) 

##开源协议

Lessgo 项目采用商业应用友好的 [MIT](https://github.com/lessgo/lessgo/raw/master/doc/LICENSE) 协议发布。
