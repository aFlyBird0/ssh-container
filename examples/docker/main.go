package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"github.com/aFlyBird0/sshcontainer/docker"
	"github.com/aFlyBird0/sshcontainer/examples/util"
)

func main() {

	// tested on Ubuntu with different docker versions

	// get ssh client
	const (
		hostPort = "1.2.3.4:22"
		user     = "root"
		pwd      = "xxx"         // pwd or key
		keyFile  = "path/to/key" // pwd or key
	)
	// the type of sshClient is *ssh.Client
	sshClient := util.CreateSSHClient(hostPort, user, pwd, keyFile)

	logrus.Infof("start to create docker client")

	// create a temporary docker socket file on local
	localSocket := "./.sock/docker.sock"
	remoteSocket := docker.DefaultDockerSock

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// create docker client with socket tunnel
	dockerClient, err := docker.NewClientWithTunnel(sshClient, localSocket, remoteSocket,
		docker.WithAutoRemoveLocalSocket,
		docker.WithLogger(logger),
		docker.WithPingRetry(10),
		docker.WithDockerClientOpts(
			client.WithTimeout(10*time.Second),
			client.WithAPIVersionNegotiation(),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	// declare that socket tunnel is no longer in use, automatically clear socket, close socket tunnel
	defer dockerClient.DoneAndWait()

	// business logic of docker client
	if err := doSomeOperations(dockerClient.Client); err != nil {
		logrus.Errorf(err.Error())
	}
}

func doSomeOperations(dockerClient *client.Client) error {
	const (
		imageName     = "nginx"
		containerName = "nginx"
	)
	logrus.Infof("start to pull image")
	// pull alpine image
	out, err := dockerClient.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}
	defer out.Close()
	// read pull output
	buf := make([]byte, 1024)
	for {
		_, err := out.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return fmt.Errorf("image pull error: %w", err)
			}
		}
	}
	logrus.Infof("pull image success")

	logrus.Infof("start to list images")
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list images: %v", err)
	}
	for _, image := range images {
		logrus.Infof("image id: %s, repo tags: %v", image.ID, image.RepoTags)
	}

	logrus.Infof("create nginx container")
	_, err = dockerClient.ContainerCreate(context.Background(),
		&container.Config{Image: "nginx"},
		nil, nil, nil, containerName)
	if err != nil {
		logrus.Errorf("Unable to create container: %v", err)
	}

	logrus.Infof("start nginx container")
	err = dockerClient.ContainerStart(context.Background(), containerName, types.ContainerStartOptions{})
	if err != nil {
		logrus.Errorf("Unable to start container: %v", err)
	}

	logrus.Infof("list containers")
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %v", err)
	}
	for _, container := range containers {
		logrus.Infof("container id: %s, name: %s", container.ID, container.Names)
	}

	return nil
}
