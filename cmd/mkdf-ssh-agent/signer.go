// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"crypto"
	"crypto/ed25519"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/tillitis/tillitis-key1-apps/internal/uss"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
	"github.com/tillitis/tillitis-key1-apps/mkdfsign"
)

var ErrMaybeWrongDevice = errors.New("wrong device or non-responsive app")

// Makefile copies the built app here ./app.bin
//
//go:embed app.bin
var appBinary []byte

const (
	// 4 chars each.
	wantAppName0 = "mkdf"
	wantAppName1 = "sign"
)

type Signer struct {
	tk         *mkdf.TillitisKey
	mkdfSigner *mkdfsign.Signer
}

func NewSigner(devPath string, speed int, enterUSS bool, fileUSS string) (*Signer, error) {
	mkdf.SilenceLogging()
	le.Printf("Connecting to device on serial port %s ...\n", devPath)
	tk, err := mkdf.New(devPath, speed)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %w", err)
	}

	mkdfSigner := mkdfsign.New(tk)
	signer := &Signer{&tk, &mkdfSigner}

	// Start handling signals here to catch abort during USS entering
	handleSignals(func() {
		if err := signer.disconnect(); err != nil {
			le.Printf("%s\n", err)
		}
		os.Exit(1)
	}, os.Interrupt, syscall.SIGTERM)

	if err = signer.maybeLoadApp(enterUSS, fileUSS); err != nil {
		// We're failing, clean up and return the more important error
		if err2 := signer.disconnect(); err2 != nil {
			le.Printf("%s\n", err2)
		}
		return nil, err
	}
	return signer, nil
}

func (s *Signer) maybeLoadApp(enterUSS bool, fileUSS string) error {
	if s.isWantedApp() {
		if enterUSS || fileUSS != "" {
			le.Printf("App is already loaded, USS flags are ignored.\n")
		}
		return nil
	} else if !s.isFirmwareMode() {
		// now we know that:
		// - loaded app does not have the wanted name
		// - device is not in firmware mode
		// anything else is possible
		return ErrMaybeWrongDevice
	}

	le.Printf("Device is in firmware mode.\n")
	var secret []byte
	var err error
	if enterUSS {
		secret, err = uss.InputUSS()
		if err != nil {
			return fmt.Errorf("Failed to get USS: %w", err)
		}
	} else if fileUSS != "" {
		secret, err = uss.ReadUSS(fileUSS)
		if err != nil {
			return fmt.Errorf("Failed to read uss-file %s: %w", fileUSS, err)
		}
	}

	if len(secret) > 0 {
		le.Printf("Loading USS...\n")
		if err = s.tk.LoadUSS(secret); err != nil {
			return fmt.Errorf("tk.LoadUSS: %w", err)
		}
	}

	le.Printf("Loading app...\n")
	if err = s.tk.LoadApp(appBinary); err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	return nil
}

func (s *Signer) disconnect() error {
	if s.mkdfSigner == nil {
		return nil
	}
	if err := s.mkdfSigner.Close(); err != nil {
		return fmt.Errorf("mkdfSigner.Close: %w", err)
	}
	return nil
}

func (s *Signer) isWantedApp() bool {
	nameVer, err := s.mkdfSigner.GetAppNameVersion()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			le.Printf("GetAppNameVersion: %s\n", err)
		}
		return false
	}
	// not caring about nameVer.Version
	if wantAppName0 != nameVer.Name0 || wantAppName1 != nameVer.Name1 {
		return false
	}
	return true
}

func (s *Signer) isFirmwareMode() bool {
	_, err := s.tk.GetNameVersion()
	return err == nil
}

// implementing crypto.Signer below

func (s *Signer) Public() crypto.PublicKey {
	pub, err := s.mkdfSigner.GetPubkey()
	if err != nil {
		le.Printf("GetPubKey failed: %s\n", err)
		return nil
	}
	return ed25519.PublicKey(pub)
}

func (s *Signer) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error) {
	// The Ed25519 signature must be made over unhashed message. See:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/crypto/ed25519/ed25519.go;l=80
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("message must not be hashed")
	}

	signature, err := s.mkdfSigner.Sign(message)
	if err != nil {
		return nil, fmt.Errorf("Sign: %w", err)
	}
	return signature, nil
}

func handleSignals(action func(), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		for {
			<-ch
			action()
		}
	}()
}
