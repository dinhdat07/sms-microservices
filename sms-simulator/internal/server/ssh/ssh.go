package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

func StartDummySSHServer(port int) error {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Accept anything, or just accept "sim"
			if c.User() == "sim" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	privateKey, err := generatePrivateKey()
	if err != nil {
		return err
	}
	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	log.Printf("Dummy SSH server listening on port %d", port)

	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil {
				log.Printf("failed to accept incoming connection: %s", err)
				continue
			}

			go handleConn(nConn, config)
		}
	}()

	return nil
}

func handleConn(nConn net.Conn, config *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return // Ignore handshakes that fail
	}

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "exec":
					if req.WantReply {
						_ = req.Reply(true, nil)
						_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					}
					_ = channel.Close()
				case "env":
					if req.WantReply {
						_ = req.Reply(true, nil)
						_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					}
				default:
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
		}(requests)
	}
}

func generatePrivateKey() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(priv)
}
