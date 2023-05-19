# SSH Container

[English](README.md) | [简体中文]

使用 SSH 访问远程主机上的容器（Docker、Containerd 等），使得在不开启容器远程访问的情况下，也能使用 SDK 操控容器。

> 注意：目前项目还在实验阶段，随时可能重构（引入 Breaking Change）。

## 为什么使用 SSH Container

当我们想在程序中操控远程的容器的时候（以使用Go语言操控Docker为例），可能会想到以下两种方式：

1. 开启 Docker 的远程访问，使用 Docker 的 Go SDK 访问
2. 使用 Go 通过 SSH 登录远程机器，然后通过 Docker CLI 访问
   
但二者都有缺点：

1. 第一种方式的缺点是不安全，且需要重启Docker Daemon（这可能会影响用户的当前业务）；第二种方式，需要使用正则提取 Docker CLI 的输出，代码又难写又不优雅。 
2. 所以，有没有一种，在不开启 Docker 远程访问的情况下，也能使用 Go SDK 来操控 Docker 以得到原生的 Docker 对象呢？（其他容器、语言也是）

是的，SSH Container 就是答案！

## 原理

我们知道，容器的 Client 端（例如通过 CLI 或者 SDK 来控制容器），本质上都是通过 Unix Domain Socket 来实现的。

所以我们可以通过 SSH 在本地和远程之间建立一个 Socket 隧道，当 Client 访问本地 Socket 实际上就等同于访问远程 Socket。

下面以 Docker 举例：

1. 我们在本地建立一个临时 socket 路径，例如 `~/docker_tmp.sock` （防止和本地的 `~/docker.sock` 冲突）
2. 通过执行 `ssh -nNT -L ~/docker.sock:/var/run/docker.sock user@ip`，将所有向 `~/docker_tmp.sock`的请求，转发到远程机器的 `/var/run/docker.sock`
3. 在终端内执行 `export DOCKER_HOST=unix://~/docker_tmp.sock`，这样，我们就可以通过 Docker CLI 访问远程机器的 Docker Daemon 了。
4. 最后，删除 `~/docker_tmp.sock` 即可。

而 SSH Container 就是对上述过程的封装，用 Go 语言实现了 `ssh -L` 等核心逻辑，以及自动关闭隧道等等。

## 使用方法（待完善）

### 1. 单纯使用隧道功能，自行创建容器 Client

详见 [`examples/tunnel/main.go`](examples/tunnel/main.go)。

### 2. 使用封装好的容器 Client（目前支持 Docker、Containerd）

本项目同时提供了对 Docker 和 Containerd 的简单的封装

* 会返回一个匿名嵌套了 Docker 或 Containerd 的 Client 的结构体）
* 会自动处理一些细节（如第一次建立连接时重试、容器连接参数的设置等）

详见：

* [`examples/docker/main.go`](examples/docker/main.go)
* [`examples/containerd/main.go`](examples/containerd/main.go)

## 致谢

* @Esonhugh 提供了转发 `docker.sock` 的核心思路。
* [sshtunnel](https://github.com/elliotchance/sshtunnel) 本项目的 tunnel 部分源自于该项目的简化和修改。
* [SSH port forwarding with Go](https://eli.thegreenplace.net/2022/ssh-port-forwarding-with-go/) 启发了我使用 Go 实现 tunnel 而不是使用 `ssh -L`。
