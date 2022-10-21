// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	_ "embed"
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
)

// Makefile copies the built app here ./app.bin
//
//go:embed app.bin
var appBinary []byte

const (
	wantAppName0 = "mkdf"
	wantAppName1 = "rand"
)

var le = log.New(os.Stderr, "", 0)

func main() {
	var devPath string
	var speed, bytes int
	pflag.StringVar(&devPath, "port", "/dev/ttyACM0", "Path to serial port device")
	pflag.IntVar(&speed, "speed", 38400, "When talking over the serial port, bits per second")
	pflag.IntVarP(&bytes, "bytes", "b", 0, "Number of random bytes to get")
	pflag.Parse()

	if bytes == 0 {
		le.Printf("Please set number of bytes with --bytes\n")
		pflag.Usage()
		os.Exit(2)
	}

	mkdf.SilenceLogging()

	le.Printf("Connecting to device on serial port %s...\n", devPath)
	tk, err := mkdf.New(devPath, speed)
	if err != nil {
		le.Printf("Could not open %s: %v\n", devPath, err)
		os.Exit(1)
	}

	randomGen := New(tk)
	exit := func(code int) {
		if err := randomGen.Close(); err != nil {
			le.Printf("%v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	if !isWantedApp(randomGen) {
		if !isFirmwareMode(tk) {
			le.Printf("If the serial port is correct for the device, then it might not be it\n" +
				"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
			exit(1)
		}
		le.Printf("Device is in firmware mode. Loading app...\n")
		if err = tk.LoadApp(appBinary, []byte{}); err != nil {
			le.Printf("LoadApp failed: %v", err)
			exit(1)
		}
	}

	le.Printf("Random data follows on stdout...\n")

	left := bytes
	for {
		get := left
		if get > RandomPayloadMaxBytes {
			get = RandomPayloadMaxBytes
		}
		random, err := randomGen.GetRandom(get)
		if err != nil {
			le.Printf("GetRandom failed: %v\n", err)
			exit(1)
		}
		if left > len(random) {
			os.Stdout.Write(random)
			left -= len(random)
			continue
		}
		os.Stdout.Write(random[0:left])
		break
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

func isWantedApp(randomGen RandomGen) bool {
	nameVer, err := randomGen.GetAppNameVersion()
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

func isFirmwareMode(tk mkdf.TillitisKey) bool {
	_, err := tk.GetNameVersion()
	return err == nil
}
