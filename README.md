# SSH Container

[English] | [简体中文](README-zh-CN.md)

Using SSH to access containers (such as Docker, Containerd, etc.) on remote hosts allows you to manipulate containers using SDK without enabling remote access to the containers.

> Note: The project is currently in an experimental phase and may undergo refactoring (including the introduction of breaking changes) at any time.

## Why SSH Container

When we want to manipulate remote containers in our program (taking Go language and Docker as an example), we may think of the following two approaches:

1. Enable remote access to Docker and use the Docker Go SDK to access it.
2. Use Go to SSH into the remote machine and then interact with Docker CLI.

However, both approaches have drawbacks:

1. The first approach is insecure and requires restarting the Docker Daemon, which can potentially impact the user's current business.
2. The second approach requires parsing the output of the Docker CLI using regular expressions, making the code complex and less elegant.

**So, is there a way to manipulate Docker using the Go SDK (or other container technologies and programming languages) without enabling remote access to Docker?**

Yes, SSH Container is the answer!

## How SSH Container Works

We know that the client side of a container (such as controlling a container via CLI or SDK) is essentially implemented through Unix Domain Socket.

Therefore, we can establish a socket tunnel between the local and remote using SSH, so that accessing the local socket is equivalent to accessing the remote socket.

Take Docker as an example:

1. We create a temporary socket path locally, for example, `~/docker_tmp.sock` (to avoid conflicts with the local `~/docker.sock`).
2. By executing `ssh -nNT -L ~/docker.sock:/var/run/docker.sock user@ip`, all requests to `~/docker_tmp.sock` will be forwarded to `/var/run/docker.sock` on the remote machine.
3. Execute `export DOCKER_HOST=unix://~/docker_tmp.sock` in the terminal. This allows us to access the remote machine's Docker Daemon via Docker CLI.
4. Finally, delete `~/docker_tmp.sock`.

SSH Container is a wrapper around the above process. It implements core logic such as `ssh -L` using Go language and automatically closes the tunnel, and so on.

## Usage (to be improved)

### 1. Using tunnel functionality only and creating a container client on your own

Refer to [`examples/tunnel/main.go`](examples/tunnel/main.go).

### 2. Using the pre-wrapped container client (currently supporting Docker and Containerd)

This project also provides simple wrappers for Docker and Containerd.

* It returns a struct that anonymously embeds the Client for Docker or Containerd.
* It automatically handles some details such as retrying on the first connection establishment and setting container connection parameters.

Refer to:

* [`examples/docker/main.go`](examples/docker/main.go)
* [`examples/containerd/main.go`](examples/containerd/main.go)


## Acknowledgments

* @Esonhugh Provided me with the core idea of forwarding `docker.sock`.
* [sshtunnel](https://github.com/elliotchance/sshtunnel) The tunnel part of this project is derived from the simplification and modification of that project.
* [SSH port forwarding with Go](https://eli.thegreenplace.net/2022/ssh-port-forwarding-with-go/) Inspired me to implement a tunnel using Go instead of using ssh -L.
