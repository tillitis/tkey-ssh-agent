package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHAgent struct {
	signer crypto.Signer
}

func NewSSHAgent(signer crypto.Signer) (*SSHAgent, error) {
	return &SSHAgent{signer: signer}, nil
}

func (s *SSHAgent) GetSSHPub() (ssh.PublicKey, error) {
	if s.signer.Public() == nil {
		return nil, fmt.Errorf("pubkey is nil")
	}
	sshPub, err := ssh.NewPublicKey(s.signer.Public())
	if err != nil {
		return nil, fmt.Errorf("NewPublicKey failed: %w", err)
	}
	return sshPub, nil
}

func (s *SSHAgent) Serve(sockPath string) error {
	sockPath, err := filepath.Abs(sockPath)
	if err != nil {
		return err
	}
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	fmt.Printf("listening on %s ...\n", sockPath)
	for {
		conn, err := listener.Accept()
		if err != nil {
			// TODO check err.Timeout() ?
			return fmt.Errorf("accept failed: %w", err)
		}
		fmt.Printf("handling connection\n")
		go s.handleConn(conn)
	}
	return nil
}

func (s *SSHAgent) handleConn(c net.Conn) {
	if err := agent.ServeAgent(s, c); !errors.Is(io.EOF, err) {
		log.Println("Agent client connection ended with error:", err)
	}
}

// implementing agent.ExtendedAgent below

func (s *SSHAgent) List() ([]*agent.Key, error) {
	sshPub, err := s.GetSSHPub()
	if err != nil {
		return nil, err
	}
	return []*agent.Key{{
		Format:  sshPub.Type(),
		Blob:    sshPub.Marshal(),
		Comment: "pubkey-of-something-hw-backed",
	}}, nil
}

var ErrNotImplemented = errors.New("not implemented")

func (s *SSHAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	sshPub, err := s.GetSSHPub()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(sshPub.Marshal(), key.Marshal()) {
		return nil, fmt.Errorf("pubkey mismatch")
	}
	sshSigner, err := ssh.NewSignerFromSigner(s.signer)
	if err != nil {
		return nil, fmt.Errorf("NewSignerFromSigner failed: %w", err)
	}
	return sshSigner.Sign(rand.Reader, data)
}

func (s *SSHAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	// we only do ed25519, so no need to care about flags
	return s.Sign(key, data)
}

func (s *SSHAgent) Extension(extensionType string, contents []byte) ([]byte, error) {
	// there is a new extensionType session-bind@openssh.com, but
	// implementation still seems optional
	// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.agent
	return nil, agent.ErrExtensionUnsupported
}

func (s *SSHAgent) Add(key agent.AddedKey) error {
	return ErrNotImplemented
}

func (s *SSHAgent) Remove(key ssh.PublicKey) error {
	return ErrNotImplemented
}

func (s *SSHAgent) RemoveAll() error {
	return ErrNotImplemented
}

func (s *SSHAgent) Lock(passphrase []byte) error {
	return ErrNotImplemented
}

func (s *SSHAgent) Unlock(passphrase []byte) error {
	return ErrNotImplemented
}

func (s *SSHAgent) Signers() ([]ssh.Signer, error) {
	return nil, ErrNotImplemented
}
