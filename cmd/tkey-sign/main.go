// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tkeyclient"
	"github.com/tillitis/tkeysign"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

// May be set to non-empty at build time to indicate that the signer
// app has been compiled with touch requirement removed.
var signerAppNoTouch string

func main() {
	var fileName, devPath string
	var speed int
	var showPubkeyOnly, verbose, helpOnly bool
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "p", false,
		"Don't sign anything, only output the public key.")
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", tkeyclient.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	pflag.Usage = func() {
		desc := fmt.Sprintf(`Usage: %[1]s [flags...] [FILE]

%[1]s communicates with the signer app running on Tillitis TKey and
makes it sign data provided in FILE (the "message"). The message can be at most
4096 bytes long. The signature made by the signer app is always output on stdout.
Exit status code is 0 if everything went well and the signature also can be
verified using the public key. Otherwise exit code is non-zero.

Alternatively, --show-pubkey can be used to only output (on stdout) the
public key of the signer app on the TKey.`, os.Args[0])
		le.Printf("%s\n\n%s", desc,
			pflag.CommandLine.FlagUsagesWrapped(86))
	}
	pflag.Parse()

	if pflag.NArg() > 0 {
		if pflag.NArg() > 1 {
			le.Printf("Unexpected argument: %s\n\n", strings.Join(pflag.Args()[1:], " "))
			pflag.Usage()
			os.Exit(2)
		}
		fileName = pflag.Args()[0]
	}

	if helpOnly {
		pflag.Usage()
		os.Exit(0)
	}

	if fileName == "" && !showPubkeyOnly {
		le.Printf("Please pass at least a message FILE, or -p.\n\n")
		pflag.Usage()
		os.Exit(2)
	}

	if fileName != "" && showPubkeyOnly {
		le.Printf("Pass only a message FILE or -p.\n\n")
		pflag.Usage()
		os.Exit(2)
	}

	if !verbose {
		tkeyclient.SilenceLogging()
	}

	if devPath == "" {
		var err error
		devPath, err = util.DetectSerialPort(true)
		if err != nil {
			os.Exit(1)
		}
	}

	tk := tkeyclient.New()
	le.Printf("Connecting to TKey on serial port %s ...\n", devPath)
	if err := tk.Connect(devPath, tkeyclient.WithSpeed(speed)); err != nil {
		le.Printf("Could not open %s: %v\n", devPath, err)
		os.Exit(1)
	}

	signer := tkeysign.New(tk)
	exit := func(code int) {
		if err := signer.Close(); err != nil {
			le.Printf("%v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	pubkey, err := signer.GetPubkey()
	if err != nil {
		le.Printf("GetPubKey failed: %v\n", err)
		exit(1)
	}
	if showPubkeyOnly {
		fmt.Printf("%x\n", pubkey)
		exit(0)
	}
	le.Printf("Public Key from TKey: %x\n", pubkey)

	message, err := os.ReadFile(fileName)
	if err != nil {
		le.Printf("Could not read %s: %v\n", fileName, err)
		os.Exit(1)
	}

	if len(message) > tkeysign.MaxSignSize {
		le.Printf("Message too long, max is %d bytes\n", tkeysign.MaxSignSize)
		exit(1)
	}

	le.Printf("Sending a %v bytes message for signing.\n", len(message))
	if signerAppNoTouch == "" {
		le.Printf("The TKey will flash green when touch is required ...\n")
	} else {
		le.Printf("WARNING! This tkey-sign and signer app is built with the touch requirement removed\n")
	}
	signature, err := signer.Sign(message)
	if err != nil {
		le.Printf("Sign failed: %v\n", err)
		exit(1)
	}
	le.Printf("Signature over message by TKey (on stdout):\n")
	fmt.Printf("%x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		le.Printf("Signature FAILED verification.\n")
		exit(1)
	}
	le.Printf("Signature verified.\n")

	exit(0)
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
