// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tillitis-key1-apps/tk1"
	"github.com/tillitis/tillitis-key1-apps/tk1sign"
)

func main() {
	fileName := pflag.String("file", "",
		"Read data to be signed (the \"message\") from `FILE`.")
	port := pflag.String("port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	speed := pflag.Int("speed", tk1.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	verbose := pflag.Bool("verbose", false,
		"Enable verbose output.")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n%s", os.Args[0],
			pflag.CommandLine.FlagUsagesWrapped(80))
	}
	pflag.Parse()

	if !*verbose {
		tk1.SilenceLogging()
	}

	if *fileName == "" {
		fmt.Printf("Please pass at least --file\n")
		pflag.Usage()
		os.Exit(2)
	}

	if *port == "" {
		var err error
		*port, err = util.DetectSerialPort(true)
		if err != nil {
			fmt.Printf("Failed to list ports: %v\n", err)
			os.Exit(1)
		} else if *port == "" {
			os.Exit(1)
		}
	}

	message, err := os.ReadFile(*fileName)
	if err != nil {
		fmt.Printf("Could not read %s: %v\n", *fileName, err)
		os.Exit(1)
	}

	tk := tk1.New()
	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	if err = tk.Connect(*port, tk1.WithSpeed(*speed)); err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}

	signer := tk1sign.New(tk)
	exit := func(code int) {
		if err = signer.Close(); err != nil {
			fmt.Printf("%v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	pubkey, err := signer.GetPubkey()
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		exit(1)
	}
	fmt.Printf("Public Key from device: %x\n", pubkey)

	fmt.Printf("Sending a %v bytes message for signing.\n", len(message))
	fmt.Printf("Device will flash green when touch is required ...\n")
	signature, err := signer.Sign(message)
	if err != nil {
		fmt.Printf("Sign failed: %v\n", err)
		exit(1)
	}
	fmt.Printf("Signature over message by device: %x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		fmt.Printf("Signature did NOT verify.\n")
		exit(1)
	} else {
		fmt.Printf("Signature verified.\n")
	}

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
