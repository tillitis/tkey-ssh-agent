// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"crypto/ed25519"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
	"github.com/tillitis/tillitis-key1-apps/mkdfsign"
)

func main() {
	fileName := pflag.String("file", "", "Name of file with data to be signed")
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", 38400, "When talking over the serial port, bits per second")
	verbose := pflag.Bool("verbose", false, "Enable verbose output")
	pflag.Parse()

	if !*verbose {
		mkdf.SilenceLogging()
	}

	message, err := os.ReadFile(*fileName)
	if err != nil {
		fmt.Printf("Could not read %s: %v\n", *fileName, err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	tk, err := mkdf.New(*port, *speed)
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}

	signer := mkdfsign.New(tk)
	exit := func(code int) {
		if err := signer.Close(); err != nil {
			fmt.Printf("%v\n", err)
		}
		os.Exit(code)
	}

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
