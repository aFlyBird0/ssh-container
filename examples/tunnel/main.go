package tunnel

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"github.com/aFlyBird0/sshcontainer/examples/util"
	"github.com/aFlyBird0/sshcontainer/tunnel"
)

func main() {
	// get ssh client
	const (
		hostPort = "1.2.3.4:22"
		user     = "root"
		pwd      = "xxx"         // pwd or key
		keyFile  = "path/to/key" // pwd or key
	)
	// the type of sshClient is *ssh.Client
	sshClient := util.CreateSSHClient(hostPort, user, pwd, keyFile)

	// create a temporary docker socket file on local
	localSocket := "~/sshcontainer/any_container.sock"
	remoteSocket := "/var/run/docker.sock" // change it to any socket path of your container
	//remoteSocket := "/run/containerd/containerd.sock"

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// create tunnel and set options
	socketTunnel := tunnel.NewSocketTunnel(localSocket, remoteSocket, sshClient).
		AutoRemoveLocalSocket().SetLogger(logger)

	// start tunnel
	go func() {
		if err := socketTunnel.Start(); err != nil {
			logger.Errorf("failed to start docker socket socketTunnel: %v", err)
			// no need to exist because docker client will retry and report error
		}
	}()

	// close tunnel when you don't need it
	defer socketTunnel.Stop()

	// business code
	dockerHost := "unix://" + localSocket
	cli, err := client.NewClientWithOpts(client.WithHost(dockerHost))
	if err != nil {
		logger.Fatalf(err.Error())
	}
	version, err := cli.ServerVersion(context.Background())
	if err != nil {
		logger.Fatalf(err.Error())
	}
	logrus.Infof("%v", version)
}
