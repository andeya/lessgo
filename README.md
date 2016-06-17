# Lessgo Web Framework  [![GoDoc](https://godoc.org/github.com/lessgo/lessgo?status.svg)](https://godoc.org/github.com/lessgo/lessgo) [![GitHub release](https://img.shields.io/github/release/lessgo/lessgo.svg)](https://github.com/lessgo/lessgo/releases)

![Lessgo Favicon](https://github.com/lessgo/doc/raw/master/img/favicon.png)

## 概述
Lessgo是一款Go语言开发的简单、稳定、高效、灵活的 web开发框架。它的项目组织形式经过精心设计，实现前后端分离、系统与业务分离，完美兼容MVC与MVVC等多种开发模式，非常利于企业级应用与API接口的开发。当然，最值得关注的是它突破性支持运行时路由重建，开发者可在Admin后台轻松配置路由，并实现启用/禁用模块或操作、添加/移除中间件等！同时，它以ApiHandler与ApiMiddleware为项目基本组成单元，可实现编译期或运行时的自由搭配组合，也令开发变得更加灵活富有趣味性。

官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

## 适用场景
- 网站
- web应用
- Restful API服务应用
- 企业应用

## 当前版本
- V0.7.0
- 发布日期：2016.06.01

## 最新功能特性
- 使用简单、运行稳定高效（核心架构来自对echo真正意义的二次开发）
- 兼容流行系统模式如:MVC、MVVC、Restful...
- httprouter真实路由配合强大的虚拟路由层，不仅性能优秀更可同时支持在源码或admin中动态配置
- 多异构数据库支持，且用户可以选择xorm或者gorm两种引擎（当然愿意，用户还可以同时使用两种引擎）
- 优化的项目目录组织最佳实践，满足复杂企业应用需要
- 集成统一的系统日志(system、database独立完整的日志)
- 提供Session管理（优化beego框架中的session包）
- 强大的前端模板渲染引擎（pongo2）
- 天生支持运行时可更新的API测试网页（swagger2.0）
- 配置文件自动补填默认值，并按字母排序
- 支持热编译
- 支持热升级
- 另外灵活的扩展包中还包含HOTP、TOTP、UUID以及各种条码生成工具等常用工具包

![Lessgo Server](https://github.com/lessgo/doc/raw/master/img/server.jpg) 
![Lessgo Server](https://github.com/lessgo/doc/raw/master/img/admin.png)

## 框架下载

```sh
go get -u github.com/lessgo/lessgo
go get -u github.com/lessgo/lessgoext/...
```

## 框架构成
- 核心框架：[lessgo](https://github.com/lessgo/lessgo)
- 框架扩展：[lessgoext](https://github.com/lessgo/lessgoext)
- 项目Demo：[demo](https://github.com/lessgo/demo)
- 框架文档  [document](https://github.com/lessgo/doc)

## 代码示例

- main.go

```go
import (
    "github.com/lessgo/lessgo"
    "github.com/lessgo/lessgoext/swagger"

    _ "github.com/lessgo/lessgoext/dbservice/xorm"
    // _ "github.com/lessgo/lessgoext/dbservice/gorm"

    _ "github.com/lessgo/demo/middleware"
    _ "github.com/lessgo/demo/router"
)

func main() {
    // 开启自动api文档，false表示仅允许内网访问
    swagger.Reg(false)
    // 指定根目录URL
    lessgo.SetHome("/home")
    // 开启网络服务
    lessgo.Run()
}
```

- 定义一个较复杂的操作

```
import (
    . "github.com/lessgo/lessgo"
    "github.com/lessgo/demo/sysmodel/admin"
)

var Index = ApiHandler{
    Desc:   "后台管理登录操作",
    Method: "POST|PUT",
    Params: []Param{
        {"user", "formData", true, "henry11111", "用户名"},
        {"user", "query", true, "henry22222", "用户名"},
        {"user", "path", true, "henry33333", "用户名"},
        {"password", "formData", true, "1111111111", "密码"},
        {"password", "query", true, "2222222222", "密码"},
        {"password", "path", true, "3333333333", "密码"},
    },
    Handler: func(c *Context) error {
        // 测试读取cookie
        id := c.CookieParam(Config.Session.SessionName)
        c.Log().Info("cookie中的%v: %#v", Config.Session.SessionName, id)

        // 测试session
        c.Log().Info("从session读取上次请求的输入: %#v", c.GetSession("info"))

        c.SetSession("info", map[string]interface{}{
            "user":     c.FormParam("user"),
            "password": c.FormParam("password"),
        })
        c.Log().Info("path用户名: %#v", c.PathParam("user"))
        c.Log().Info("query用户名: %#v", c.QueryParam("user"))
        c.Log().Info("formData用户名: %#v", c.FormParam("user"))
        c.Log().Info("path密码: %#v", c.PathParam("password"))
        c.Log().Info("query密码: %#v", c.QueryParam("password"))
        c.Log().Info("formData密码: %#v", c.FormParam("password"))

        return c.Render(200,
            "sysview/admin/login/index.tpl",
            map[string]interface{}{
                "name":       c.FormParam("user"),
                "password":   c.FormParam("password"),
                "repeatfunc": admin.Login.Repeatfunc,
            },
        )
    },
}.Reg()
```

- 一个简单的数据模型

```
import (
    "strings"
)
type login struct{}
var Login = login{}

func (_ login) Repeatfunc(s string, count int) string {
    return strings.Repeat(s, count)
}
```

- 一个简单的中间件

```
var ShowHeader = lessgo.ApiMiddleware{
    Name:   "显示Header",
    Desc:   "显示Header测试",
    Config: nil,
    Middleware: func(c *lessgo.Context) error {
        c.Log().Info("测试中间件-显示Header：%v", c.Request().Header)
        return nil
    },
}.Reg()
```

- 在源码中定义路由

```go
package router

import (
    "github.com/lessgo/lessgo"

    "github.com/lessgo/demo/bizhandler/home"
    "github.com/lessgo/demo/middleware"
)

func init() {
    lessgo.Root(
        lessgo.Leaf("/websocket", home.WebSocket, middleware.ShowHeader),
        lessgo.Branch("/home", "前台",
            lessgo.Leaf("/index", home.Index, middleware.ShowHeader),
        ).Use(middleware.Print),
    )
}
```

## 系统文档
- [综述](https://github.com/lessgo/doc/blob/master/Introduction.md)
- [安装部署](https://github.com/lessgo/doc/blob/master/Install.md)
- [开始lessgo之旅](https://github.com/lessgo/doc/blob/master/Develop01.md)
- [更多(文档目录)](https://github.com/lessgo/doc/blob/master/README.md)

## 项目架构
![Lessgo Web Framework](https://github.com/lessgo/doc/raw/master/img/LessgoWebFramework.jpg)


## 项目目录结构

```
─Project 项目开发目录
├─config 配置文件目录
│  ├─app.config 系统应用配置文件
│  └─db.config 数据库配置文件
├─common 后端公共目录
│  └─... 如utils等其他
├─middleware 后端公共中间件目录
├─static 前端公共目录 (url: /static)
│  ├─tpl 公共tpl模板目录
│  ├─js 公共js目录 (url: /static/js)
│  ├─css 公共css目录 (url: /static/css)
│  ├─img 公共img目录 (url: /static/img)
│  └─plugin 公共js插件 (url: /static/plugin)
├─uploads 默认上传下载目录
├─router 源码路由配置
│  ├─sysrouter.go 系统模块路由文件
│  ├─bizrouter.go 业务模块路由文件
├─syshandler 系统模块后端目录
│  ├─xxx 子模块目录
│  │  ├─example.go example操作
│  │  └─... xxx的子模块目录
│  └─... 其他子模块目录
├─sysmodel 系统模块数据模型目录
├─sysview 系统模块前端目录 (url: /sys)
│  ├─xxx 与syshandler对应的子模块目录 (url: /sys/xxx)
│  │  ├─example.tpl 相应操作的模板文件
│  │  ├─example2.html 无需绑定操作的静态html文件
│  │  ├─xxx.css css文件(可有多个)
│  │  ├─xxx.js js文件(可有多个)
│  │  └─... xxx的子模块目录
├─bizhandler 业务模块后端目录
│  ├─xxx 子模块目录
│  │  ├─example.go example操作
│  │  └─... xxx的子模块目录
│  └─... 其他子模块目录
├─bizmodel 业务模块数据模型目录
├─bizview 业务模块前端目录 (url: /biz)
│  ├─xxx 与bizhandler对应的子模块目录 (url: /biz/xxx)
│  │  ├─example.tpl 相应操作的模板文件
│  │  ├─example2.html 无需绑定操作的静态html文件
│  │  ├─xxx.css css文件(可有多个)
│  │  ├─xxx.js js文件(可有多个)
│  │  └─... xxx的子模块目录
├─database 默认数据库文件存储目录
├─logger 运行日志输出目录
└─main.go 应用入口文件
```

## 贡献者名单
贡献者                          |贡献概要
--------------------------------|--------------------------------------------------
[henrylee2cn](https://github.com/henrylee2cn)|代码的主要实现者 (第一作者) 
[changyu72](https://github.com/changyu72)|架构的主要设计者 (第二作者) 
[LeSou](https://github.com/LeSou)|

## 开源协议
Lessgo 项目采用商业应用友好的 [MIT](https://github.com/lessgo/lessgo/raw/master/LICENSE) 协议发布。
