// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHAgent struct {
	signer *Signer
	mutex  sync.Mutex
}

func NewSSHAgent(signer *Signer) *SSHAgent {
	return &SSHAgent{signer: signer}
}

func (s *SSHAgent) GetAuthorizedKey() ([]byte, error) {
	sshPub, err := s.getSSHPub()
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(sshPub), nil
}

func (s *SSHAgent) getSSHPub() (ssh.PublicKey, error) {
	pub := s.signer.Public()
	if pub == nil {
		return nil, fmt.Errorf("pubkey is nil")
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("NewPublicKey: %w", err)
	}
	return sshPub, nil
}

func (s *SSHAgent) Serve(absSockPath string) error {
	listener, err := net.Listen("unix", absSockPath)
	if err != nil {
		return fmt.Errorf("Listen: %w", err)
	}
	le.Printf("listening on %s ...\n", absSockPath)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Accept: %w", err)
		}
		le.Printf("handling a client connection\n")
		go s.handleConn(conn)
	}
}

func (s *SSHAgent) handleConn(c net.Conn) {
	if err := agent.ServeAgent(s, c); !errors.Is(io.EOF, err) {
		le.Printf("Agent client connection ended with error: %s\n", err)
	}
}

// implementing agent.ExtendedAgent below

func (s *SSHAgent) List() ([]*agent.Key, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sshPub, err := s.getSSHPub()
	if err != nil {
		return nil, err
	}
	udi, err := s.signer.GetUDI()
	if err != nil {
		return nil, err
	}
	return []*agent.Key{{
		Format:  sshPub.Type(),
		Blob:    sshPub.Marshal(),
		Comment: fmt.Sprintf("TK1-%s", udi),
	}}, nil
}

var ErrNotImplemented = errors.New("not implemented")

func (s *SSHAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	le.Printf("Sign: user will have to touch the device\n")
	sshPub, err := s.getSSHPub()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(sshPub.Marshal(), key.Marshal()) {
		return nil, fmt.Errorf("pubkey mismatch")
	}
	sshSigner, err := ssh.NewSignerFromSigner(s.signer)
	if err != nil {
		return nil, fmt.Errorf("NewSignerFromSigner: %w", err)
	}
	signature, err := sshSigner.Sign(rand.Reader, data)
	if err != nil {
		return nil, fmt.Errorf("Signer.Sign: %w", err)
	}
	return signature, nil
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
