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
	"go.bug.st/serial"
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

type MKDFSigner struct {
	devPath string
	conn    serial.Port
	speed   int
}

func NewMKDFSigner(devPath string, speed int) (*MKDFSigner, error) {
	mkdf.SilenceLogging()
	signer := &MKDFSigner{
		devPath: devPath,
		speed:   speed,
	}
	le.Printf("Connecting to device on serial port %s ...\n", devPath)
	if err := signer.connect(); err != nil {
		return nil, err
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

func (s *MKDFSigner) connect() error {
	var err error
	s.conn, err = serial.Open(s.devPath, &serial.Mode{BaudRate: s.speed})
	if err != nil {
		return fmt.Errorf("Could not open %s: %w", s.devPath, err)
	}
	return nil
}

func (s *MKDFSigner) disconnect() error {
	if s.conn == nil {
		return nil
	}
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("Close: %w", err)
	}
	return nil
}

func (s *MKDFSigner) isFirmwareMode() bool {
	_, err := mkdf.GetNameVersion(s.conn)
	return err == nil
}

func (s *MKDFSigner) isWantedApp() bool {
	nameVer, err := mkdfsign.GetAppNameVersion(s.conn)
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

func (s *MKDFSigner) loadApp(bin []byte) error {
	if err := mkdf.LoadApp(s.conn, bin); err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	return nil
}

// implementing crypto.Signer below

func (s *MKDFSigner) Public() crypto.PublicKey {
	pub, err := mkdfsign.GetPubkey(s.conn)
	if err != nil {
		le.Printf("GetPubKey failed: %s\n", err)
		return nil
	}
	return ed25519.PublicKey(pub)
}

func (s *MKDFSigner) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error) {
	// The Ed25519 signature must be made over unhashed message. See:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/crypto/ed25519/ed25519.go;l=80
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("message must not be hashed")
	}

	signature, err := mkdfsign.Sign(s.conn, message)
	if err != nil {
		return nil, fmt.Errorf("Sign: %w", err)
	}
	return signature, nil
}
