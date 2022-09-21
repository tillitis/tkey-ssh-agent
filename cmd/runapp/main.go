// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
	"go.bug.st/serial"
)

func main() {
	fileName := pflag.String("file", "", "App binary to be uploaded and started")
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", 38400, "When talking over the serial port, bits per second")
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
	conn, err := serial.Open(*port, &serial.Mode{BaudRate: *speed})
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}
	exit := func(code int) {
		conn.Close()
		os.Exit(code)
	}

	nameVer, err := mkdf.GetNameVersion(conn)
	if err != nil {
		fmt.Printf("GetNameVersion failed: %v\n", err)
		fmt.Printf("If the serial port device is correct, then the device might not be in\n" +
			"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
		exit(1)
	}
	fmt.Printf("Firmware has name0:%s name1:%s version:%d\n",
		nameVer.Name0, nameVer.Name1, nameVer.Version)
	fmt.Printf("Loading app from %v onto device\n", *fileName)
	err = mkdf.LoadAppFromFile(conn, *fileName)
	if err != nil {
		fmt.Printf("LoadAppFromFile failed: %v\n", err)
		exit(1)
	}

	exit(0)
}
