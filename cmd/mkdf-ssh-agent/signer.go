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
	*mkdf.TillitisKey
	*mkdfsign.Signer
}

func NewSigner(devPath string, speed int) (*Signer, error) {
	mkdf.SilenceLogging()
	le.Printf("Connecting to device on serial port %s ...\n", devPath)
	tk, err := mkdf.New(devPath, speed)
	if err != nil {
		return nil, fmt.Errorf("Could not open %s: %w", devPath, err)
	}
	s := mkdfsign.New(tk)
	signer := &Signer{
		TillitisKey: &tk,
		Signer:      &s,
	}
	if !signer.isWantedApp() {
		if !signer.isFirmwareMode() {
			// now we know that:
			// - loaded app does not have the wanted name
			// - device is not in firmware mode
			// anything else is possible
			return nil, ErrMaybeWrongDevice
		}
		le.Printf("Device is in firmware mode, loading app...\n")
		if err := signer.loadApp(appBinary); err != nil {
			return nil, err
		}
	}
	return signer, nil
}

func (s *Signer) disconnect() error {
	if s.Signer == nil {
		return nil
	}
	if err := s.Signer.Close(); err != nil {
		return fmt.Errorf("signer.Close: %w", err)
	}
	return nil
}

func (s *Signer) isFirmwareMode() bool {
	_, err := s.GetNameVersion()
	return err == nil
}

func (s *Signer) isWantedApp() bool {
	nameVer, err := s.Signer.GetAppNameVersion()
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

func (s *Signer) loadApp(bin []byte) error {
	if err := s.LoadApp(bin); err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	return nil
}

// implementing crypto.Signer below

func (s *Signer) Public() crypto.PublicKey {
	pub, err := s.Signer.GetPubkey()
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

	signature, err := s.Signer.Sign(message)
	if err != nil {
		return nil, fmt.Errorf("Sign: %w", err)
	}
	return signature, nil
}
