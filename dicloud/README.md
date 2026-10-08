# 信创版本项目（DiCloud）

<!--采集云+业务云-->

## 一、项目

### 1.包含项

​	项目包含采集云，和业务云。整合原来的两个系统。

### 2.项目目标

​	项目目标，旨在原项目（采集云，业务云）完成目标任务后；朝向新国家战略背景，优化原项目结构，使用信创版本系统、中间件实现新版本系统。

### 3.开发语言

​	Go语言！！！

​	为适配多平台部署，便于迁移各种类型机器；采用Go语言开发，编译之后便于移植。适合小型项目/机器部署。

### 4.技术选型

#### 	SQL数据库

​		[**OpenGauss**](https://opengauss.org/zh/)

​		华为数据库，和MySQL之间可以进行无缝切换。

#### 	消息中间件

​		[**RocketMQ**](https://rocketmq.apache.org/)

​		阿里巴巴开发开源消息中间件。

#### 	时序数据库

​		[**TDengine**](https://www.taosdata.com/)

​		涛思数据库，提供时序数据存储服务。

#### 	代码管理工具

​		[**Gitee码云**](https://gitee.com/)

​		国内易用代码管理工具。

### 5.目录结构

​	参考：[微服务架构下， Go Web 项目目录结构怎么组织最合理 – PingCode](https://docs.pingcode.com/ask/ask-ask/313223.html)

​	[【软件工具】GO语言目录结构、模块简介_go 目录结构-CSDN博客](https://blog.csdn.net/weixin_46453070/article/details/135677930)

​	[Go整洁架构设计——目录结构基本布局_go项目目录结构最佳-CSDN博客](https://blog.csdn.net/hguisu/article/details/129342554)

​	其他：https://github.com/kubernetes/kubernetes/

​	https://github.com/gogs/gogs

> 如：

```xml
- cmd			// 用于存放应用程序的入口点，每个微服务应该有一个子目录，其中包含主应用程序的入口文件。
  - auth
  - business
- configs
- docs
- internal		// 用于存放本微服务特定的代码，这些代码不应该被其他微服务访问，也不应该被直接导入。
  - auth
  - business
- models		// 用于存放与数据库交互的模型和数据访问逻辑。
- pkg			// 用于存放可供其他微服务直接导入的代码，通常是一些工具函数、通用中间件等。
- scripts		// 脚本文件，执行各种构建，安装，分析等操作的脚本。
- test			// 单元测试之外的测试程序，测试数据。
- third_party
- util			// 工具包。
.gitignore
go.mod
go.sum
README.md

```



目录介绍：

在微服务架构下，采用以下目录结构组织Go Web项目：

- `cmd`：用于存放应用程序的入口点，每个微服务应该有一个子目录，其中包含主应用程序的入口文件。
- `configs`：用于存放微服务的配置文件，包括数据库连接信息、环境变量等。
- `internal`：用于存放本微服务特定的代码，这些代码不应该被其他微服务访问，也不应该被直接导入。
  - `handlers`（或者`controllers`）：用于存放与HTTP请求相关的处理逻辑，包括路由处理函数、HTTP中间件等。
  - `services`：用于存放微服务的业务逻辑，包括对数据的处理、组织、验证等。
  - `dto`：用于存放与前端交互的模型，对数据库数据的再封装。
- `models`（或者`repositories`）：用于存放与数据库交互的模型和数据访问逻辑。
- `pkg`：用于存放可供其他微服务直接导入的代码，通常是一些工具函数、通用中间件等。
- `logs`：用于存放微服务的日志文件。
- `script`：脚本文件，执行各种构建，安装，分析等操作的脚本。
- `test`：单元测试之外的测试程序，测试数据。
- `util`（或者`tools`）：工具包。



> 如：`auth`模块mudule示例

```xml
- cmd
  - auth
	auth.go
	routes.go
  - business
- configs
- docs
- internal
  - auth
	- controller
	- dto
	- model
	- service
  - business
- models
- pkg
  - common
  - middleware
- scripts
- test
- third_party
- util
.gitignore
go.mod
go.sum
README.md

```



<img src="./docs/programing_md_files/prj_struct.png"/>



### 6.项目模块

#### 采集云

#### 业务云

#### 公共部分



## 二、首要任务

### 1.实现基础功能

- 采集云：`mqtt`模块，`deal`模块，`api`模块，`tolink`模块。
- 业务云：`auth`模块，`business`模块，`alarm`模块。



## 三、编译/部署

> Windows系统

### 1.安装Go环境

参考：[Go 菜鸟教程](https://www.runoob.com/go/go-ide.html)

### 2.设置编译环境

```bash
set GOOS=linux
```

### 3.编译

```bash
go build auth.go routes.go
```

### 4.部署

- 创建项目运行根目录
- 上传文件
- 设置文件为可执行文件
- 运行文件
- 可用守护工具管理运行项目

