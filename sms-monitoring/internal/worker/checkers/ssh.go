package checkers

import (
	"context"
	"fmt"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/security"

	"golang.org/x/crypto/ssh"
)

type SSHChecker struct {
	timeout time.Duration
}

func NewSSHChecker(timeout time.Duration) HealthChecker {
	return &SSHChecker{timeout: timeout}
}

func (c *SSHChecker) Check(ctx context.Context, config ServerConfig) bool {
	ip := config["ipv4"]
	port := config["ssh_port"]
	user := config["ssh_user"]
	key := config["ssh_key"]
	if key != "" {
		if decrypted, err := security.Decrypt(key); err == nil {
			key = decrypted
		} else {
			logger.Log.Sugar().Errorf("[SSHChecker] Failed to decrypt ssh key for IP %s: %v", ip, err)
			return false
		}
	}
	if ip == "" || port == "" || user == "" || key == "" {
		return false
	}

	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		logger.Log.Sugar().Errorf("[SSHChecker] Failed to parse private key for IP %s: %v", ip, err)
		return false
	}

	authMethod := ssh.PublicKeys(signer)

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.timeout,
	}

	addr := fmt.Sprintf("%s:%s", ip, port)

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		logger.Log.Sugar().Errorf("[SSHChecker] Failed to dial %s: %v", addr, err)
		return false
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	if err := session.Run("echo 1"); err != nil {
		return false
	}

	return true
}
