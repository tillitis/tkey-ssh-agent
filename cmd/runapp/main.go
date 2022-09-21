// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/mkdf"
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

	tk, err := mkdf.New(*port, *speed)
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}
	exit := func(code int) {
		tk.Close()
		os.Exit(code)
	}

	nameVer, err := tk.GetNameVersion()
	if err != nil {
		fmt.Printf("GetNameVersion failed: %v\n", err)
		fmt.Printf("If the serial port device is correct, then the device might not be in\n" +
			"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
		exit(1)
	}
	fmt.Printf("Firmware has name0:%s name1:%s version:%d\n",
		nameVer.Name0, nameVer.Name1, nameVer.Version)
	fmt.Printf("Loading app from %v onto device\n", *fileName)
	err = tk.LoadAppFromFile(*fileName)
	if err != nil {
		fmt.Printf("LoadAppFromFile failed: %v\n", err)
		exit(1)
	}

	exit(0)
}
