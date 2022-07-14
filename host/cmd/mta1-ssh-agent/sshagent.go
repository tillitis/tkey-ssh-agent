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

type sshAgent struct {
	signer crypto.Signer
	sshPub ssh.PublicKey
}

func NewSshAgent(signer crypto.Signer) (*sshAgent, error) {
	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		return nil, fmt.Errorf("fetch pubkey failed: %w", err)
	}
	s := &sshAgent{
		signer: signer,
		sshPub: sshPub,
	}
	return s, nil
}

func (s *sshAgent) serve(sockPath string) error {
	sockPath, err := filepath.Abs(sockPath)
	if err != nil {
		return err
	}
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen failed: %w\n", err)
	}
	fmt.Printf("listening on %s ...\n", sockPath)
	for {
		conn, err := listener.Accept()
		if err != nil {
			// TODO check err.Timeout() ?
			return fmt.Errorf("accept failed: %w\n", err)
		}
		fmt.Printf("handling connection\n")
		go s.handleConn(conn)
	}
	return nil
}

func (s *sshAgent) handleConn(c net.Conn) {
	if err := agent.ServeAgent(s, c); err != io.EOF {
		log.Println("Agent client connection ended with error:", err)
	}
}

// implementing agent.ExtendedAgent below

func (s *sshAgent) List() ([]*agent.Key, error) {
	return []*agent.Key{{
		Format:  s.sshPub.Type(),
		Blob:    s.sshPub.Marshal(),
		Comment: "pubkey-of-something-hw-backed",
	}}, nil
}

var ErrNotImplemented = errors.New("not implemented")

func (s *sshAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	if !bytes.Equal(s.sshPub.Marshal(), key.Marshal()) {
		return nil, fmt.Errorf("pubkey mismatch")
	}
	sshSigner, err := ssh.NewSignerFromSigner(s.signer)
	if err != nil {
		return nil, fmt.Errorf("signerfromsigner failed: %w", err)
	}
	signature, err := sshSigner.Sign(rand.Reader, data)
	return signature, err
}

func (s *sshAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	// we only do ed25519, so no need to care about flags
	return s.Sign(key, data)
}

func (s *sshAgent) Extension(extensionType string, contents []byte) ([]byte, error) {
	// there is a new extensionType session-bind@openssh.com, but
	// implementation still seems optional
	// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.agent
	return nil, agent.ErrExtensionUnsupported
}

func (s *sshAgent) Add(key agent.AddedKey) error {
	return ErrNotImplemented
}

func (s *sshAgent) Remove(key ssh.PublicKey) error {
	return ErrNotImplemented
}

func (s *sshAgent) RemoveAll() error {
	return ErrNotImplemented
}

func (s *sshAgent) Lock(passphrase []byte) error {
	return ErrNotImplemented
}

func (s *sshAgent) Unlock(passphrase []byte) error {
	return ErrNotImplemented
}

func (s *sshAgent) Signers() ([]ssh.Signer, error) {
	return nil, ErrNotImplemented
}
