package tunnel

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/aFlyBird0/sshcontainer/log"
)

const unix = "unix"

type SocketTunnel struct {
	localSocket           string
	remoteSocket          string
	autoRemoveLocalSocket bool
	log                   log.Logger

	sshClient *ssh.Client
	conns     []net.Conn
	close     chan struct{}
	isOpen    bool
	done      chan struct{}
	listener  net.Listener
}

func NewSocketTunnel(localSocket, remoteSocket string, sshClient *ssh.Client) *SocketTunnel {
	return &SocketTunnel{
		localSocket:  localSocket,
		remoteSocket: remoteSocket,
		log:          logrus.New(),
		sshClient:    sshClient,
		close:        make(chan struct{}, 1),
		done:         make(chan struct{}, 1),
	}
}

func (tunnel *SocketTunnel) SetLogger(logger log.Logger) *SocketTunnel {
	tunnel.log = logger
	return tunnel
}

func (tunnel *SocketTunnel) AutoRemoveLocalSocket() *SocketTunnel {
	tunnel.autoRemoveLocalSocket = true
	return tunnel
}

func (tunnel *SocketTunnel) DisableLogger() *SocketTunnel {
	tunnel.log = &log.NoopLogger{}
	return tunnel
}

func (tunnel *SocketTunnel) Start() (err error) {
	// mkdir -p if not exists
	if err = os.MkdirAll(filepath.Dir(tunnel.localSocket), 0755); err != nil {
		return fmt.Errorf("failed to create local socket directory: %v", err)
	}
	if err = tunnel.removeLocalSocket(); err != nil {
		return err
	}

	tunnel.log.Debugf("starting tunnel from %s to %s\n", tunnel.localSocket, tunnel.remoteSocket)
	tunnel.listener, err = net.Listen(unix, tunnel.localSocket)
	if err != nil {
		return fmt.Errorf("failed to listen on local socket: %v", err)
	}

	defer tunnel.listener.Close()

	defer func() {
		total := len(tunnel.conns)
		for i, conn := range tunnel.conns {
			tunnel.log.Debugf("closing the netConn (%d of %d)\n", i+1, total)
			err := conn.Close()
			if err != nil && !errors.Is(err, net.ErrClosed) {
				tunnel.log.Errorf("failed to close netConn: %v\n", err)
			}
		}
	}()

	tunnel.isOpen = true

	for tunnel.isOpen {
		c := make(chan net.Conn)
		go tunnel.newConnectionWaiter(tunnel.listener, c)

		select {
		case <-tunnel.close:
			// 接收到中断信号，终止程序
			tunnel.log.Debugf("received close signal\n")
			tunnel.isOpen = false
		case conn := <-c:
			tunnel.conns = append(tunnel.conns, conn)
			tunnel.log.Debugf("accepted connection\n")

			go func() {
				err := tunnel.forward(conn)
				if err != nil {
					tunnel.log.Errorf("failed to forward connection: %v\n", err)
				}
			}()
		}
	}

	/// ensure all connections are closed
	tunnel.done <- struct{}{}

	return nil
}

func (tunnel *SocketTunnel) forward(local net.Conn) error {
	// Issue a dial to the remote server on our SSH client; here "localhost"
	// refers to the remote server.
	remote, err := tunnel.sshClient.Dial(unix, tunnel.remoteSocket)
	if err != nil {
		return fmt.Errorf("failed to dial remote socket: %v", err)
	}

	runTunnel(local, remote)
	return nil
}

func (tunnel *SocketTunnel) Stop() {
	tunnel.close <- struct{}{}
	close(tunnel.close)

	if err := tunnel.removeLocalSocket(); err != nil {
		tunnel.log.Errorf("failed to remove local socket file: %v", err)
	}

	// ensure all connections are closed
	<-tunnel.done
}

func (tunnel *SocketTunnel) newConnectionWaiter(listener net.Listener, c chan net.Conn) {
	tunnel.log.Debugf("waiting for new connection\n")
	conn, err := listener.Accept()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		tunnel.log.Errorf("failed to accept connection: %v\n", err)
		return
	}
	c <- conn
}

func runTunnel(local, remote net.Conn) {
	defer local.Close()
	defer remote.Close()
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()

	<-done
}

// remove localSocket if exists
func (tunnel *SocketTunnel) removeLocalSocket() error {
	if !tunnel.autoRemoveLocalSocket {
		return nil
	}
	if _, err := os.Stat(tunnel.localSocket); err == nil {
		if err := os.Remove(tunnel.localSocket); err != nil {
			return fmt.Errorf("failed to remove local socket file: %v", err)
		}
	}
	return nil
}
