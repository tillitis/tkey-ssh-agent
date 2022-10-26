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
	"github.com/tillitis/tillitis-key1-apps/tk1"
	"github.com/tillitis/tillitis-key1-apps/tk1sign"
)

func main() {
	fileName := pflag.String("file", "", "Name of file with data to be signed (the \"message\")")
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", tk1.SerialSpeed, "When talking over the serial port, bits per second")
	verbose := pflag.Bool("verbose", false, "Enable verbose output")
	pflag.Parse()

	if !*verbose {
		tk1.SilenceLogging()
	}

	if *fileName == "" {
		fmt.Printf("Please pass at least --file\n")
		pflag.Usage()
		os.Exit(2)
	}

	message, err := os.ReadFile(*fileName)
	if err != nil {
		fmt.Printf("Could not read %s: %v\n", *fileName, err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	tk, err := tk1.New(*port, *speed)
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}

	signer := tk1sign.New(tk)
	exit := func(code int) {
		if err := signer.Close(); err != nil {
			fmt.Printf("%v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	udi, err := signer.GetUDI()
	if err != nil {
		fmt.Printf("GetUDI failed: %v\n", err)
		exit(1)
	}

	pubkey, err := signer.GetPubkey()
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		exit(1)
	}
	fmt.Printf("Public Key from device (UID %v): %x\n", udi, pubkey)

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
