package util

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func CreateSSHClient(hostPort, user, pwd, keyFile string) *ssh.Client {
	logrus.Infof("start to dial ssh")
	var auths []ssh.AuthMethod
	if keyFile != "" {
		if keyAuth, err := authenticationKey(keyFile); err == nil {
			auths = append(auths, keyAuth)
		} else {
			logrus.Warnf("unable to use key file (%s): %v", keyFile, err)
		}
	}
	if pwd != "" {
		auths = append(auths, ssh.Password(pwd))
	}

	if len(auths) == 0 {
		logrus.Fatalf("no authentication method")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ignore host key check
		Timeout:         10 * time.Second,
	}

	// connect to ssh
	sshClient, err := ssh.Dial("tcp", hostPort, config)
	if err != nil {
		log.Fatalf("Failed to dial SSH: %v", err)
	}

	return sshClient
}

// get private key from private key file path
func authenticationKey(privateKeyPath string) (ssh.AuthMethod, error) {
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}

	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return ssh.PublicKeys(privateKey), nil
}
