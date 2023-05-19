package containerd

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/aFlyBird0/sshcontainer/log"
	"github.com/aFlyBird0/sshcontainer/tunnel"
)

const DefaultContainerdSocket = "/run/containerd/containerd.sock"

// ClientWithTunnel is containerd client with tunnel
type ClientWithTunnel struct {
	*containerd.Client
	socketTunnel   *tunnel.SocketTunnel
	containerdOpts []containerd.ClientOpt
	maxRetry       uint
	log            log.Logger
}

// Opt is option for ClientWithTunnel
type Opt func(*ClientWithTunnel) error

// NewClientWithTunnel create containerd client with tunnel
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
			c.log.Errorf("failed to start containerd socket tunnel: %v", err)
			// no need to exit because containerd client will retry and report error
		}
	}()

	socketPath := localSocket
	c.log.Debugf("socketPath: %s", socketPath)

	cl, err := containerd.New(socketPath, c.containerdOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %v", err)
	}
	c.Client = cl

	// try to connect to containerd socket
	if err := c.pingWithRetry(); err != nil {
		return nil, err
	}

	return c, nil
}

// WithMaxRetry set max retry for connecting to containerd socket
func (c *ClientWithTunnel) pingWithRetry() error {
	for i := uint(0); i < c.maxRetry; i++ {
		if i != 0 {
			time.Sleep(1 * time.Second)
		}

		// todo: namespace
		ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
		if _, err := c.Client.HealthService().Check(ctx, &grpc_health_v1.HealthCheckRequest{}, grpc.WaitForReady(true)); err == nil {
			c.log.Debugf("connected to containerd socket")
			return nil
		}

		c.log.Debugf("failed to connect to containerd socket, retrying...")
	}

	return fmt.Errorf("failed to connect to containerd socket")
}

// DoneAndWait stop tunnel and wait for it to exit
func (c *ClientWithTunnel) DoneAndWait() {
	c.socketTunnel.Stop()
}

// WithLogger set logger for ClientWithTunnel
func WithLogger(log log.Logger) Opt {
	return func(c *ClientWithTunnel) error {
		c.log = log
		return nil
	}
}

// WithAutoRemoveLocalSocket will remove local socket when tunnel exit
func WithAutoRemoveLocalSocket(c *ClientWithTunnel) error {
	c.socketTunnel.AutoRemoveLocalSocket()
	return nil
}

// WithDisableLogger disable all log output
func WithDisableLogger(c *ClientWithTunnel) error {
	c.log = &log.NoopLogger{}
	return nil
}

// WithContainerdClientOpts set origin containerd client options
func WithContainerdClientOpts(opts ...containerd.ClientOpt) Opt {
	return func(c *ClientWithTunnel) error {
		c.containerdOpts = opts
		return nil
	}
}

// WithPingRetry set max retry for connecting to containerd socket, default is 3
func WithPingRetry(maxRetry uint) Opt {
	return func(c *ClientWithTunnel) error {
		c.maxRetry = maxRetry
		return nil
	}
}
