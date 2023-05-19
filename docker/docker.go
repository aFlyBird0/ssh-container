package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"

	"github.com/aFlyBird0/sshcontainer/log"
	"github.com/aFlyBird0/sshcontainer/tunnel"
)

const DefaultDockerSock = "/var/run/docker.sock"

// ClientWithTunnel is docker client with tunnel
type ClientWithTunnel struct {
	*client.Client

	socketTunnel *tunnel.SocketTunnel

	dockerOpts []client.Opt
	maxRetry   uint
	log        log.Logger
}

// Opt is option for ClientWithTunnel
type Opt func(*ClientWithTunnel) error

// NewClientWithTunnel create docker client with tunnel
func NewClientWithTunnel(sshClient *ssh.Client, localSocket, remoteSocket string, opts ...Opt) (*ClientWithTunnel, error) {
	tunnel := tunnel.NewSocketTunnel(localSocket, remoteSocket, sshClient)
	c := &ClientWithTunnel{
		socketTunnel: tunnel,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.log == nil {
		c.log = &log.NoopLogger{}
	}
	c.socketTunnel.SetLogger(c.log)
	if c.maxRetry == 0 {
		c.maxRetry = 3
	}

	go func() {
		if err := tunnel.Start(); err != nil {
			c.log.Errorf("failed to start docker socket tunnel: %v", err)
			// no need to exist because docker client will retry and report error
		}
	}()

	dockerHost := "unix://" + localSocket
	c.dockerOpts = append(c.dockerOpts, client.WithHost(dockerHost))

	cli, err := client.NewClientWithOpts(c.dockerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %v", err)
	}
	c.Client = cli

	// try to connect to docker socket
	if err := c.pingWithRetry(); err != nil {
		return nil, err
	}

	return c, nil
}

// PingWithRetry try to ping docker socket with retry to make sure it's ready
func (c *ClientWithTunnel) pingWithRetry() error {
	for i := uint(0); i < c.maxRetry; i++ {
		if i != 0 {
			time.Sleep(1 * time.Second)
		}

		if _, err := c.Ping(context.Background()); err == nil {
			c.log.Debugf("connected to docker socket")
			return nil
		}

		c.log.Debugf("failed to connect to docker socket, retrying...")
	}

	return fmt.Errorf("failed to connect to docker socket")
}

// DoneAndWait stop tunnel and wait for all connections closed
func (c *ClientWithTunnel) DoneAndWait() {
	c.socketTunnel.Stop()
}

// WithLogger set custom logger
func WithLogger(log log.Logger) Opt {
	return func(c *ClientWithTunnel) error {
		c.log = log
		return nil
	}
}

// WithAutoRemoveLocalSocket remove local socket file before and after tunnel
func WithAutoRemoveLocalSocket(c *ClientWithTunnel) error {
	c.socketTunnel.AutoRemoveLocalSocket()
	return nil
}

func WithDisableLogger(c *ClientWithTunnel) error {
	c.log = &log.NoopLogger{}
	return nil
}

// WithDockerClientOpts set original docker client options
func WithDockerClientOpts(opts ...client.Opt) Opt {
	return func(c *ClientWithTunnel) error {
		c.dockerOpts = opts
		return nil
	}
}

// WithPingRetry set max retry times to connect to docker socket, default is 3
func WithPingRetry(maxRetry uint) Opt {
	return func(c *ClientWithTunnel) error {
		c.maxRetry = maxRetry
		return nil
	}
}
