// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
	"golang.org/x/term"
)

func main() {
	fileName := pflag.String("file", "", "App binary to be uploaded and started")
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", 38400, "When talking over the serial port, bits per second")
	typeUSS := pflag.Bool("uss", false, "Enable typing of a phrase for the User Supplied Secret. The phrase\n"+
		"is hashed using BLAKE2 to a digest. The USS digest is used by the\n"+
		"firmware, together with other material, for deriving secrets for the\n"+
		"application.")
	verbose := pflag.Bool("verbose", false, "Enable verbose output")
	pflag.Parse()

	if !*verbose {
		mkdf.SilenceLogging()
	}

	if *fileName == "" {
		fmt.Printf("Please pass at least --file\n")
		pflag.Usage()
		os.Exit(2)
	}

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)

	tk, err := mkdf.New(*port, *speed)
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}
	exit := func(code int) {
		if err := tk.Close(); err != nil {
			fmt.Printf("Close: %v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	nameVer, err := tk.GetNameVersion()
	if err != nil {
		fmt.Printf("GetNameVersion failed: %v\n", err)
		fmt.Printf("If the serial port device is correct, then the device might not be in\n" +
			"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
		exit(1)
	}
	fmt.Printf("Firmware has name0:%s name1:%s version:%d\n",
		nameVer.Name0, nameVer.Name1, nameVer.Version)

	if *typeUSS {
		uss, err := inputUSS()
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			exit(1)
		}
		fmt.Printf("Loading USS onto device\n")
		if err = tk.LoadUSS(uss); err != nil {
			fmt.Printf("LoadUSS failed: %v\n", err)
			exit(1)
		}
	}

	fmt.Printf("Loading app from %v onto device\n", *fileName)
	err = tk.LoadAppFromFile(*fileName)
	if err != nil {
		fmt.Printf("LoadAppFromFile failed: %v\n", err)
		exit(1)
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

func inputUSS() ([]byte, error) {
	fmt.Printf("Enter phrase for the USS: ")
	uss, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("ReadPassword: %w", err)
	}
	fmt.Printf("\nRepeat the phrase: ")
	ussAgain, err := term.ReadPassword(0)
	if err != nil {
		return nil, fmt.Errorf("ReadPassword: %w", err)
	}
	fmt.Printf("\n")
	if bytes.Compare(uss, ussAgain) != 0 {
		return nil, fmt.Errorf("phrases did not match")
	}
	if len(uss) == 0 {
		return nil, fmt.Errorf("no phrase entered")
	}
	return uss, nil
}
