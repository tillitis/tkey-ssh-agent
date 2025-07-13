// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"crypto"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tillitis/tkeyclient"
	"github.com/tillitis/tkeysign"
	"github.com/tillitis/tkeyutil"
	"golang.org/x/crypto/ssh"
)

var notify = func(msg string) {
	tkeyutil.Notify(progname, msg)
}

const (
	idleDisconnect = 3 * time.Second
	// 4 chars each.
	wantFWName0  = "tk1 "
	wantFWName1  = "mkdf"
	wantAppName0 = "tk1 "
	wantAppName1 = "sign"
)

type Signer struct {
	tk              *tkeyclient.TillitisKey
	tkSigner        *tkeysign.Signer
	port            Port
	uss             UssConfig
	mu              sync.Mutex
	connected       bool
	disconnectTimer *time.Timer
}

func NewSigner(port Port, uss UssConfig, exitFunc func(int)) *Signer {
	var signer Signer

	tkeyclient.SilenceLogging()

	tk := tkeyclient.New()

	tkSigner := tkeysign.New(tk)
	signer = Signer{
		tk:       tk,
		tkSigner: &tkSigner,
		port:     port,
		uss:      uss,
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

	if s.disconnectTimer != nil {
		s.disconnectTimer.Stop()
		s.disconnectTimer = nil
	}

	if s.connected {
		return true
	}

	devPath := s.port.Path
	if devPath == "" {
		var err error
		devPath, err = tkeyclient.DetectSerialPort(false)
		if err != nil {
			switch {
			case errors.Is(err, tkeyclient.ErrNoDevice):
				notify("Could not find any TKey plugged in.")
			case errors.Is(err, tkeyclient.ErrManyDevices):
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
	if err := s.tk.Connect(devPath, tkeyclient.WithSpeed(s.port.Speed)); err != nil {
		notify(fmt.Sprintf("Could not connect to a TKey on port %v.", devPath))
		le.Printf("Failed to connect: %v", err)
		return false
	}

	if s.isFirmwareMode() {
		le.Printf("TKey is in firmware mode.\n")

		udi, err := s.tk.GetUDI()
		if err != nil {
			le.Printf("Failed to get UDI: %v\n", err)
			s.closeNow()
			return false
		}

		app, err := GetApp(udi.ProductID)
		if err != nil {
			notify("Uknown product ID. Failed to identify what device app to use.")
			s.closeNow()

			return false
		}

		if err := s.loadApp(app, *udi); err != nil {
			le.Printf("Failed to load app: %v\n", err)
			s.closeNow()
			return false
		}
	}

	if !s.isWantedApp() {
		// Notifying because we're kinda stuck if we end up here
		notify("Please remove and plug in your TKey again\n— it might be running the wrong app.")
		le.Printf("No TKey on the serial port, or it's running wrong app (and is not in firmware mode)")
		s.closeNow()
		return false
	}

	// We nowadays disconnect from the TKey when idling, so the
	// signer-app that's running may have been loaded by somebody
	// else. Therefore we can never be sure it has USS according to
	// the flags that tkey-ssh-agent was started with. So we no longer
	// say anything about that.

	s.connected = true
	return true
}

func (s *Signer) isFirmwareMode() bool {
	nameVer, err := s.tk.GetNameVersion()
	if err != nil {
		return false
	}
	// not caring about nameVer.Version
	return nameVer.Name0 == wantFWName0 &&
		nameVer.Name1 == wantFWName1
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
	return nameVer.Name0 == wantAppName0 &&
		nameVer.Name1 == wantAppName1
}

func (s *Signer) loadApp(devApp []byte, udi tkeyclient.UDI) error {
	var secret []byte
	var err error

	if s.uss.EnterManually {
		secret, err = getSecret(udi.String(), s.uss.PinentryPath)
		if err != nil {
			notify(fmt.Sprintf("Could not show USS prompt: %s", errors.Unwrap(err)))
			return fmt.Errorf("Failed to get USS: %w", err)
		}
	} else if s.uss.Path != "" {
		var err error
		secret, err = tkeyutil.ReadUSS(s.uss.Path)
		if err != nil {
			notify(fmt.Sprintf("Could not read USS file: %s", err))
			return fmt.Errorf("Failed to read uss-file %s: %w", s.uss.Path, err)
		}
	}

	le.Printf("Loading signer app...\n")
	if err := s.tk.LoadApp(devApp, secret); err != nil {
		return fmt.Errorf("LoadApp: %w", err)
	}
	le.Printf("Signer app loaded.\n")

	return nil
}

func (s *Signer) printAuthorizedKey() {
	if !s.connect() {
		le.Printf("Connect failed")
		return
	}
	defer s.disconnect()

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
		s.disconnectTimer = nil
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

func (s *Signer) Sign(_ io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error) {
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
