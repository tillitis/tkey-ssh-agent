// Copyright (C) 2022, 2023 - Tillitis AB
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
	"sync"
	"syscall"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tillitis-key1-apps/tk1"
	"github.com/tillitis/tillitis-key1-apps/tk1sign"
	"golang.org/x/crypto/ssh"
)

// nolint:typecheck // Avoid lint error when the embedding file is missing.
// Makefile copies the built app here ./app.bin
//
//go:embed app.bin
var appBinary []byte

const (
	idleDisconnect = 3 * time.Second
	// 4 chars each.
	wantAppName0 = "tk1 "
	wantAppName1 = "sign"
)

type Signer struct {
	tk              *tk1.TillitisKey
	tkSigner        *tk1sign.Signer
	devPath         string
	speed           int
	enterUSS        bool
	fileUSS         string
	pinentry        string
	mu              sync.Mutex
	connected       bool
	disconnectTimer *time.Timer
}

func NewSigner(devPathArg string, speedArg int, enterUSS bool, fileUSS string, pinentry string, exitFunc func(int)) *Signer {
	var signer Signer

	tk1.SilenceLogging()

	tk := tk1.New()

	tkSigner := tk1sign.New(tk)
	signer = Signer{
		tk:       tk,
		tkSigner: &tkSigner,
		devPath:  devPathArg,
		speed:    speedArg,
		enterUSS: enterUSS,
		fileUSS:  fileUSS,
		pinentry: pinentry,
	}

	// Do nothing on HUP, in case old udev rule is still in effect
	handleSignals(func() {}, syscall.SIGHUP)

	// Start handling signals here to catch abort during USS entering
	handleSignals(func() {
		signer.closeNow()
		exitFunc(1)
	}, os.Interrupt, syscall.SIGTERM)

	return &signer
}

func (s *Signer) connect() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		if s.disconnectTimer != nil {
			s.disconnectTimer.Stop()
			s.disconnectTimer = nil
		}
		return true
	}

	devPath := s.devPath
	if devPath == "" {
		var err error
		devPath, err = util.DetectSerialPort(false)
		if err != nil {
			switch {
			case errors.Is(err, util.ErrNoDevice):
				notify("Could not find any TKey plugged in.")
			case errors.Is(err, util.ErrManyDevices):
				notify("Cannot work with more than 1 TKey plugged in.")
			default:
				notify(fmt.Sprintf("TKey detection failed: %s\n", err))
			}
			le.Printf("Failed to detect port: %v\n", err)
			return false
		}
		le.Printf("Auto-detected serial port %s\n", devPath)
	}

	le.Printf("Connecting to TKey on serial port %s\n", devPath)
	if err := s.tk.Connect(devPath, tk1.WithSpeed(s.speed)); err != nil {
		notify(fmt.Sprintf("Failed to connect to a TKey on port %v.", devPath))
		le.Printf("Failed to connect: %v", err)
		return false
	}

	if !s.isWantedApp() {
		// Note: we're just assuming it's firmware if we get any reply
		_, err := s.tk.GetNameVersion()
		if err != nil {
			// Notifying because we're kinda stuck if we end up here
			notify("Please remove and plug in your TKey again\n— it might be running the wrong app.")
			le.Printf("No TKey on the serial port, or it's not in firmware mode (and already running wrong app)")
			s.closeNow()
			return false
		}
		le.Printf("The TKey is in firmware mode.\n")

		if err = s.loadApp(); err != nil {
			le.Printf("Failed to load app: %v\n", err)
			s.closeNow()
			return false
		}
	} else {
		if s.enterUSS || s.fileUSS != "" {
			le.Printf("Signer app already loaded, USS flags are ignored.\n")
		} else {
			le.Printf("Signer app already loaded.\n")
		}
	}

	s.printAuthorizedKey()

	s.connected = true
	return true
}

func (s *Signer) isWantedApp() bool {
	nameVer, err := s.tkSigner.GetAppNameVersion()
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

func (s *Signer) loadApp() error {
	var secret []byte
	if s.enterUSS {
		udi, err := s.tk.GetUDI()
		if err != nil {
			return fmt.Errorf("Failed to get UDI: %w", err)
		}

		secret, err = getSecret(udi.String(), s.pinentry)
		if err != nil {
			return fmt.Errorf("Failed to get USS: %w", err)
		}
	} else if s.fileUSS != "" {
		var err error
		secret, err = util.ReadUSS(s.fileUSS)
		if err != nil {
			return fmt.Errorf("Failed to read uss-file %s: %w", s.fileUSS, err)
		}
	}

	le.Printf("Loading signer app...\n")
	if err := s.tk.LoadApp(appBinary, secret); err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	le.Printf("Signer app loaded.\n")

	return nil
}

func (s *Signer) printAuthorizedKey() {
	pub, err := s.tkSigner.GetPubkey()
	if err != nil {
		le.Printf("GetPubkey failed: %s\n", err)
		return
	}

	sshPub, err := ssh.NewPublicKey(ed25519.PublicKey(pub))
	if err != nil {
		le.Printf("NewPublicKey failed: %s\n", err)
		return
	}

	le.Printf("Your SSH public key (on stdout):\n")
	fmt.Fprintf(os.Stdout, "%s", ssh.MarshalAuthorizedKey(sshPub))
}

func (s *Signer) disconnect() {
	if s.tkSigner == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return
	}

	if s.disconnectTimer != nil {
		s.disconnectTimer.Stop()
	}

	s.disconnectTimer = time.AfterFunc(idleDisconnect, func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.closeNow()
		s.connected = false
		s.disconnectTimer = nil
		le.Printf("Disconnected from TKey\n")
	})
}

func (s *Signer) closeNow() {
	if s.tkSigner == nil {
		return
	}
	if err := s.tkSigner.Close(); err != nil {
		le.Printf("Close failed: %s\n", err)
	}
}

// implementing crypto.Signer below

func (s *Signer) Public() crypto.PublicKey {
	if !s.connect() {
		return nil
	}
	defer s.disconnect()

	pub, err := s.tkSigner.GetPubkey()
	if err != nil {
		le.Printf("GetPubkey failed: %s\n", err)
		return nil
	}
	return ed25519.PublicKey(pub)
}

func (s *Signer) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error) {
	if !s.connect() {
		return nil, fmt.Errorf("Connect failed")
	}
	defer s.disconnect()

	// The Ed25519 signature must be made over unhashed message. See:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/crypto/ed25519/ed25519.go;l=80
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("message must not be hashed")
	}

	signature, err := s.tkSigner.Sign(message)
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

func notify(msg string) {
	if err := beeep.Notify(progname, msg, ""); err != nil {
		le.Printf("Notify message %q failed: %s\n", msg, err)
	}
}
