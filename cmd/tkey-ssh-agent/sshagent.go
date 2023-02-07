// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

//go:build windows
// +build windows

package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// May be set to non-empty at build time to indicate that the signer
// app has been compiled with touch requirement removed.
var signerAppNoTouch string

type SSHAgent struct {
	signer      *Signer
	operationMu sync.Mutex // only handling 1 agent op at a time
}

func NewSSHAgent(signer *Signer) *SSHAgent {
	return &SSHAgent{signer: signer}
}

func (s *SSHAgent) Serve(absSockPath string) error {
	pipePath := `\\.\\pipe\\tkey-ssh-agent`

	if err := os.RemoveAll(pipePath); err != nil {
		le.Printf("error: %v", err)
	}

	pc := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;AU)",
		InputBufferSize:    4096,
		OutputBufferSize:   4096,
	}

	l, err := winio.ListenPipe(pipePath, pc)

	if err != nil {
		fmt.Println("listen error:", err)
	}

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
		}
		fmt.Println("got a connection - dispatching to handler")
		go s.handleConn(conn)
	}
	/*
		listener, err := net.Listen("unix", absSockPath)
		if err != nil {
			return fmt.Errorf("Listen: %w", err)
		}
		le.Printf("Listening on %s\n", absSockPath)
		for {
			if err != nil {
				return fmt.Errorf("Accept: %w", err)
			}
			le.Printf("Handling a client connection\n")
			go s.handleConn(l)
		}
	*/
}

func (s *SSHAgent) handleConn(c net.Conn) {
	if err := agent.ServeAgent(s, c); !errors.Is(io.EOF, err) {
		le.Printf("Agent client connection ended with error: %s\n", err)
	}
}

// implementing agent.ExtendedAgent below

var ErrNotImplemented = errors.New("not implemented")

func (s *SSHAgent) List() ([]*agent.Key, error) {
	s.operationMu.Lock()
	defer s.operationMu.Unlock()

	// Connect early to be able to return empty list if that fails
	if !s.signer.connect() {
		le.Printf("List: connect failed, returning empty list\n")
		return []*agent.Key{}, nil
	}

	pub := s.signer.Public()
	if pub == nil {
		return nil, fmt.Errorf("pubkey is nil")
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("NewPublicKey: %w", err)
	}

	return []*agent.Key{{
		Format:  sshPub.Type(),
		Blob:    sshPub.Marshal(),
		Comment: "TKey",
	}}, nil
}

func (s *SSHAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	s.operationMu.Lock()
	defer s.operationMu.Unlock()

	// This does s.signer.Public()
	sshSigner, err := ssh.NewSignerFromSigner(s.signer)
	if err != nil {
		return nil, fmt.Errorf("NewSignerFromSigner: %w", err)
	}

	if !bytes.Equal(key.Marshal(), sshSigner.PublicKey().Marshal()) {
		return nil, fmt.Errorf("pubkey mismatch")
	}

	if signerAppNoTouch == "" {
		timer := time.AfterFunc(4*time.Second, func() {
			notify("Touch your TKey to confirm SSH login.")
		})
		defer timer.Stop()

		le.Printf("Sign: user will have to touch the TKey\n")
	} else {
		le.Printf("Sign: WARNING! This tkey-ssh-agent and signer app is built with the touch requirement removed\n")
	}
	signature, err := sshSigner.Sign(rand.Reader, data)
	if err != nil {
		return nil, fmt.Errorf("Signer.Sign: %w", err)
	}
	return signature, nil
}

func (s *SSHAgent) SignWithFlags(key ssh.PublicKey, data []byte, _ agent.SignatureFlags) (*ssh.Signature, error) {
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
