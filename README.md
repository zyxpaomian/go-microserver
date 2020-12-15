# MicroServer
___
 
自用golang的Server 框架，插件式，消息通过protobuf 进行序列化/反序列化
 
 
### msg消息体生成
> 消息体使用protobuf 封装, 原文件在protobuf下，msg需要手动生成
```shell
protoc --go_out=./msg/ protobuf/agent.proto
mv msg/protobuf/agent.pb.go msg/
rm -rf msg/protobuf
```

### 安装
``` shell
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
go env -w GO111MODULE=on

# 运行
make run 

# 编译
make compile
```
 
### 主要目录结构
* common 基本库，如日志，mysql 驱动, 字符转换，配置加载等
* controller 控制层，控制agent基本的逻辑代码位置，如listagent, delagent等
* dao dao层，数据层
* http mux+http封装的http 层，接口封装
* msg 消息结构体
* plugin 插件代码位置
* structs 统一的结构体位置
