package main

import (
	"context"
	"fmt"
	"time"

	ctrd "github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/sirupsen/logrus"

	"github.com/aFlyBird0/sshcontainer/containerd"
	"github.com/aFlyBird0/sshcontainer/examples/util"
)

// to be tested
func main() {
	// todo: test it

	// get ssh client
	const (
		hostPort = "1.2.3.4:22"
		user     = "root"
		pwd      = "xxx"         // pwd or key
		keyFile  = "path/to/key" // pwd or key
	)
	// the type of sshClient is *ssh.Client
	sshClient := util.CreateSSHClient(hostPort, user, pwd, keyFile)

	logrus.Infof("start to create containerd client")

	localSocket := "./.sock/containerd.sock"
	remoteSocket := containerd.DefaultContainerdSocket

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	containerdClient, err := containerd.NewClientWithTunnel(sshClient, localSocket, remoteSocket,
		containerd.WithAutoRemoveLocalSocket,
		containerd.WithLogger(logger),
		containerd.WithPingRetry(10),
		containerd.WithContainerdClientOpts(
			ctrd.WithTimeout(10*time.Second),
			// todo namespace
			ctrd.WithDefaultNamespace("k8s.io")))

	if err != nil {
		logrus.Errorf("Failed to create containerd client: %v\n", err)
	}

	// declare that socket tunnel is no longer in use, automatically clear socket, close socket tunnel
	defer containerdClient.DoneAndWait()

	// business logic of containerd client
	if err := doSomeOperations(containerdClient.Client); err != nil {
		logrus.Errorf(err.Error() + "\n")
	}
}

func doSomeOperations(client *ctrd.Client) error {
	const (
		imageName     = "docker.io/library/nginx:latest"
		containerName = "nginx"
	)

	logrus.Infof("start to pull image")
	// pull alpine image
	_, err := client.Pull(context.Background(), imageName, ctrd.WithPullUnpack)
	if err != nil {
		logrus.Errorf("Unable to pull image: %v", err)
	}
	logrus.Infof("pull image success")

	logrus.Infof("start to list images")
	images, err := client.ImageService().List(context.Background())
	if err != nil {
		return fmt.Errorf("Unable to list images: %v", err)
	}
	for _, image := range images {
		logrus.Infof("image id: %s, repo tags: %v", image.Name, image.Labels)
	}

	// create nginx container
	logrus.Infof("create nginx container")

	container, err := client.NewContainer(context.Background(),
		containerName, ctrd.WithImageName(imageName),
		ctrd.WithRuntime("io.containerd.runtime.v1.linux", nil),
		ctrd.WithNewSpec())
	if err != nil {
		return fmt.Errorf("Unable to create container: %v", err)
	}

	// fixme: failed to create task: mkdir /var/run/containerd: permission denied
	task, err := container.NewTask(
		context.Background(),
		cio.NewCreator(cio.WithStdio),
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}
	defer task.Delete(context.Background())

	// start nginx container
	logrus.Infof("start nginx container")
	err = task.Start(context.Background())
	if err != nil {
		return fmt.Errorf("Unable to start container: %v", err)
	}

	statusC, err := task.Wait(context.Background())
	if err != nil {
		return fmt.Errorf("failed to wait task: %v", err)
	}

	status := <-statusC
	logrus.Infof("task status: %v\n", status)

	// list containers
	logrus.Infof("list containers")
	containers, err := client.ContainerService().List(context.Background())
	if err != nil {
		return fmt.Errorf("Unable to list containers: %v", err)
	}
	for _, container := range containers {
		logrus.Infof("container id: %s, name: %s", container.ID, container.Labels)
	}

	return nil
}
