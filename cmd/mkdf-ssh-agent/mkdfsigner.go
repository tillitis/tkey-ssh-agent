package main

import (
	"crypto"
	"crypto/ed25519"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
	"github.com/tarm/serial"
)

var ErrMaybeWrongDevice = errors.New("wrong device or non-responsive app")

// Makefile copies the built app here ./app.bin
//go:embed app.bin
var appBinary []byte

var (
	// 4 chars each.
	wantAppName0 = "mkdf"
	wantAppName1 = "sign"
)

type MKDFSigner struct {
	devPath string
	port    *serial.Port
}

func NewMKDFSigner(devPath string) (*MKDFSigner, error) {
	mkdf.SilenceLogging()
	signer := &MKDFSigner{
		devPath: devPath,
	}
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
		fmt.Printf("Device is in firmware mode, loading app...\n")
		if err := signer.loadApp(appBinary); err != nil {
			return nil, err
		}
	}
	return signer, nil
}

func (s *MKDFSigner) connect() error {
	var err error
	s.port, err = serial.OpenPort(&serial.Config{
		Name:        s.devPath,
		Baud:        1_000_000,
		ReadTimeout: 3 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("OpenPort: %w", err)
	}
	return nil
}

func (s *MKDFSigner) isFirmwareMode() bool {
	_, err := mkdf.GetNameVersion(s.port)
	if err != nil {
		return false
	}
	return true
}

func (s *MKDFSigner) isWantedApp() bool {
	nameVer, err := mkdf.GetAppNameVersion(s.port)
	if err != nil {
		return false
	}
	// not caring about nameVer.Version
	if wantAppName0 != nameVer.Name0 || wantAppName1 != nameVer.Name1 {
		return false
	}
	return true
}

func (s *MKDFSigner) loadApp(bin []byte) error {
	err := mkdf.LoadApp(s.port, bin)
	if err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	return nil
}

// implementing crypto.Signer below

func (s *MKDFSigner) Public() crypto.PublicKey {
	pub, err := mkdf.GetPubkey(s.port)
	if err != nil {
		log.Printf("mkdf.GetPubKey failed: %v", err)
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

	signature, err := mkdf.Sign(s.port, message)
	if err != nil {
		log.Printf("mkdf.Sign: %v", err)
		return nil, fmt.Errorf("mkdf.Sign: %w", err)
	}
	return signature, nil
}
