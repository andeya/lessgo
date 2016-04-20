#框架下载
- 下载框架源码与框架扩展源码
```sh
go get -u github.com/lessgo/lessgo
go get -u github.com/lessgo/lessgoext
```
- 下载第三方依赖包，解压至 GOPATH/src 目录下 [[点击下载 ZIP]](https://github.com/lessgo/dependency/archive/master.zip)

#安装部署工具
```sh
cd %GOPATH%/github.com/lessgo/lessgoext/lessgo
go install
```
#创建项目
利用项目构建工具自动创建新项目(在项目目录下运行cmd)
```sh
$ lessgo new appname
```

#运行项目
以热编译模式运行（在项目目录下运行cmd）
```sh
$ cd appname
$ lessgo run
```
运行后在浏览器打开http://localhost:8080 即可看到项目首页.