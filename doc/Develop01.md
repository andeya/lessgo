#开始Lessgo之旅
```sh
$ lessgo new appname
```
##1. 项目构建完成后，创建的项目目录如下：  
─Project 项目目录    
├─BusinessAPI 业务模块后端目录  
│  ├─BusRouter.go 业务模块路由文件  
│  ├─BusCommon Business公共目录  
│  │  ├─Middleware 中间件目录  
│  │  └─Model 数据模型  
│  │  └─... 其他  
│  ├─Home模块目录  
│  │  ├─IndexHandle.go Index操作  
│  │  ├─IndexModel.go Index数据模型及模板函数  
│  └─... 创建要开发的模块目录(后端代码)  
├─BusinessView 业务模块前端目录 (url: /bus)  
│  ├─Home模块目录 (url:/)  
│  │  ├─index.tpl ExampleHandle对应的模板文件    
│  │  ├─index.html 无需绑定操作的静态html文件  
│  │  ├─index.css css文件  
│  │  ├─index.js js文件  
│  └─...创建要开发的模块目录(前端代码)   
├─Common 后端公共目录  
│  ├─Middleware 中间件目录  
│  └─Model 数据模型  
├─Config 配置文件目录  
│  ├─app.config 系统应用配置文件  
│  └─db.config 数据库配置文件
├─Logger 运行日志输出目录  
├─Static 前端公共目录 (url: /static)  
│  ├─Tpl 公共tpl模板目录  
│  ├─Js 公共js目录 (url: /static/js)  
│  ├─Css 公共css目录 (url: /static/css)  
│  ├─Img 公共img目录 (url: /static/img)  
│  └─Plugin 公共js插件 (url: /static/plugin)  
├─Swagger 系统API文档自动构建工具  
├─SystemAPI 系统模块后端目录  
│  ├─SysRouter.go 系统模块路由文件  
│  ├─SysCommon 后端公共目录  
│  │  ├─Middleware 中间件目录  
│  │  └─Model 数据模型  
│  │  └─... 其他  
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
├─Uploads 默认上传下载目录  
└─Main.go 应用入口文件 

##2.开发流程
- 规划好要开发的业务模块功能(分析设计阶段工作)
- 在/BusinessAPI/下建立本模块后端目录(模块还可以进一步细分为子模块，依次可以递归建立)； 
- 在/BusinessView/下建立本模块前端目录(模块还可以进一步细分为子模块，依次可以递归建立)，目录层次与本模块后端目录对应；
- 在本模块后端目录下建立代码文件(一般分为模块Handle.go与模块Model.go)，编写服务端代码；
- 在/BusinessAPI/SysRouter.go(业务模块路由)注册开发完成的handle的路由；
- 编写服务端gp文件对应的test文件进行单元测试；
- web应用在本模块前端目录下开发前端的html、tpl、js、css等代码，包括调用服务端API功能的代码，移动端app应用则通过相关工具开发应用调用开发的服务端API；
- lessgo run 进行模块代码运行调试。
  
