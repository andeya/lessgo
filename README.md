#Lessgo Web Framework  [![GoDoc](https://godoc.org/github.com/lessgo/lessgo?status.svg)](https://godoc.org/github.com/lessgo/lessgo) [![GitHub release](https://img.shields.io/github/release/lessgo/lessgo.svg)](https://github.com/lessgo/lessgo/releases)

![Lessgo Favicon](https://github.com/lessgo/doc/raw/master/img/favicon.png)

##概述
Lessgo是一款Go语言开发的简单、稳定、高效、灵活的 web开发框架。它的项目组织形式经过精心设计，实现前后端分离、系统与业务分离，完美兼容MVC与MVVC等多种开发模式，非常利于企业级应用与API接口的开发。当然，最值得关注的是它突破性支持运行时路由重建，开发者可在Admin后台轻松配置路由，并实现启用/禁用模块或操作、添加/移除中间件等！同时，它以ApiHandler与ApiMiddleware为项目基本组成单元，可实现编译期或运行时的自由搭配组合，也令开发变得更加灵活富有趣味性。

官方QQ群：Go-Web 编程 42730308    [![Go-Web 编程群](http://pub.idqqimg.com/wpa/images/group.png)](http://jq.qq.com/?_wv=1027&k=fzi4p1)

##适用场景
- 网站
- web应用
- Restful API服务应用
- 企业应用

##当前版本
- V0.6.0
- 发布日期：2016.05.17

##最新功能特性
- 使用简单、运行稳定高效（核心架构来自echo的真正意义的二次开发）
- 兼容流行系统模式如:MVC、MVVC、Restful...
- 强大的运行时动态路由，同时支持在源码或admin中配置（动态路由保存在数据库中）
- 多异构数据库支持（master分支使用xorm，dev-a分支使用gorm）
- 优化的项目目录组织最佳实践，满足复杂企业应用需要
- 集成统一的系统日志(system、database独立完整的日志)
- 提供Session管理（优化beego框架中的session包）
- 多种Token生成方式
- 强大的前端模板渲染引擎（pongo2）
- 天生支持运行时可更新的API测试网页（swagger2.0）
- 配置文件自动补填默认值，并按字母排序
- 支持热编译
- 支持热升级

![Lessgo Server](https://github.com/lessgo/doc/raw/master/img/server.jpg) 
![Lessgo Server](https://github.com/lessgo/doc/raw/master/img/admin.png)

##框架下载

```sh
go get -u github.com/lessgo/lessgo
go get -u github.com/lessgo/lessgoext
```

##框架构成
- 核心框架：[lessgo](https://github.com/lessgo/lessgo)
- 框架扩展：[lessgoext](https://github.com/lessgo/lessgoext)
- 项目Demo：[demo](https://github.com/lessgo/demo)
- 框架文档  [document](https://github.com/lessgo/doc)

##代码示例

- Main.go
```go
import (
    "github.com/lessgo/lessgo"
    "github.com/lessgo/lessgoext/swagger"
    _ "github.com/lessgo/demo/BusinessAPI"
    _ "github.com/lessgo/demo/Common/Middleware"
    _ "github.com/lessgo/demo/SystemAPI"
)

func main() {
    swagger.Init()
    lessgo.SetHome("/home")
    lessgo.Run()
}
```

- 定义一个较复杂的操作
```
import (
    . "github.com/lessgo/lessgo"
)

var IndexHandle = ApiHandler{
    Desc:   "后台管理登录操作",
    Method: "GET",
    Params: []Param{
        {"user", "path", true, "henry", "用户名"},
        {"password", "path", true, "12345678", "密码"},
    },
    Handler: func(ctx Context) error {
        // 测试读取cookie
        id, err := ctx.Request().Cookie("name")
        ctx.Logger().Info("cookie中的%v: %#v (%v)", "name", id, err)

        // 测试session
        ctx.Logger().Info("从session读取上次请求的输入: %#v", ctx.GetSession("info"))

        ctx.SetSession("info", map[string]interface{}{
            "user":     ctx.Param("user"),
            "password": ctx.Param("password"),
        })

        return ctx.Render(200,
            "SystemView/Admin/Login/index.tpl",
            map[string]interface{}{
                "name":       ctx.Param("user"),
                "password":   ctx.Param("password"),
                "repeatfunc": repeatfunc,
            },
        )
    },
}.Reg()
```

- 一个简单的中间件
```
var ShowHeaderWare = lessgo.RegMiddleware(lessgo.ApiMiddleware{
    Name:          "显示Header",
    Desc:          "显示Header测试",
    DefaultConfig: nil,
    Middleware: func(ctx lessgo.Context) error {
        logs.Info("测试中间件-显示Header：%v", ctx.Request().Header)
        return nil
    },
})
```

- 在源码中定义路由
```go
import (
    "github.com/lessgo/lessgo"
    "github.com/lessgo/demo/BusinessAPI/Home"
    "github.com/lessgo/demo/Common/Middleware"
)

func init() {
    lessgo.Root(
        lessgo.Leaf("/websocket", Home.WebSocketHandle, Middleware.ShowHeaderWare),
        lessgo.Branch("/home", "前台",
            lessgo.Leaf("/index", Home.IndexHandle, Middleware.ShowHeaderWare),
        ).Use(Middleware.PrintWare),
    )
}
```

##系统文档
- [综述](https://github.com/lessgo/doc/blob/master/Introduction.md)
- [安装部署](https://github.com/lessgo/doc/blob/master/Install.md)
- [开始lessgo之旅](https://github.com/lessgo/doc/blob/master/Develop01.md)
- [更多(文档目录)](https://github.com/lessgo/doc/blob/master/README.md)

##项目架构
![Lessgo Web Framework](https://github.com/lessgo/doc/raw/master/img/LessgoWebFramework.jpg)


##项目目录组织
─Project 项目开发目录  
├─Config 配置文件目录  
│  ├─app.config 系统应用配置文件  
│  └─db.config 数据库配置文件  
├─Common 后端公共目录  
│  ├─Middleware 中间件目录  
│  └─Model 数据模型  
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
├─SystemView 系统模块前端目录 (url: /sys)  
│  ├─Xxx Xxx子模块目录 (url: /sys/xxx)  
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
├─BusinessView 业务模块前端目录 (url: /bus)  
│  ├─Xxx Xxx子模块目录 (url: /bus/xxx)  
│  │  ├─example.tpl ExampleHandle对应的模板文件    
│  │  ├─example2.html 无需绑定操作的静态html文件  
│  │  ├─xxx.css css文件(可有多个)  
│  │  ├─xxx.js js文件(可有多个)  
│  │  └─... Xxx的子模块目录  
├─Uploads 默认上传下载目录  
├─Logger 运行日志输出目录  
└─Main.go 应用入口文件 

##贡献者名单
贡献者                          |贡献概要
--------------------------------|--------------------------------------------------
[henrylee2cn](https://github.com/henrylee2cn)|第一作者 (主要代码实现者) 
[changyu72](https://github.com/changyu72)|第二作者 (主要架构设计者) 

##开源协议
Lessgo 项目采用商业应用友好的 [MIT](https://github.com/lessgo/lessgo/raw/master/LICENSE) 协议发布。

