// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/spf13/pflag"
	"go.bug.st/serial"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

func main() {
	syscall.Umask(0o077)

	exit := func(code int) {
		os.Exit(code)
	}

	var sockPath, devPath string
	var speed int
	var showPubkeyOnly, listPortsOnly bool
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.StringVarP(&sockPath, "agent-socket", "a", "", "Path to bind agent's UNIX domain socket at")
	pflag.BoolVarP(&listPortsOnly, "list-ports", "", false, "List possible serial ports for --port")
	pflag.StringVar(&devPath, "port", "/dev/ttyACM0", "Path to serial port device")
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "k", false, "Don't start the agent, just output the ssh-ed25519 pubkey")
	pflag.IntVar(&speed, "speed", 38400, "When talking over the serial port, bits per second")

	pflag.Parse()

	if listPortsOnly {
		if err := listPorts(); err != nil {
			le.Printf("Failed to list ports: %v\n", err)
			exit(1)
		}
		exit(0)
	}

	if showPubkeyOnly && sockPath != "" {
		le.Printf("Can't combine -a and -k.\n\n")
		pflag.Usage()
		exit(2)
	}

	if !showPubkeyOnly && sockPath == "" {
		le.Printf("Please pass at least -a or -k.\n\n")
		pflag.Usage()
		exit(2)
	}

	if sockPath != "" {
		_, err := os.Stat(sockPath)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			le.Printf("Socket path %s exists?\n", sockPath)
			exit(1)
		}
	}

	signer, err := NewMKDFSigner(devPath, speed)
	if err != nil {
		if errors.Is(err, ErrMaybeWrongDevice) {
			le.Printf("If the serial port is correct for the device, then it might not be it\n" +
				"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
		} else {
			le.Printf("%s\n", err)
		}
		exit(1)
	}

	exit = func(code int) {
		if err := signer.disconnect(); err != nil {
			le.Printf("%s\n", err)
		}
		os.Exit(code)
	}

	agent := NewSSHAgent(signer)

	authorizedKey, err := agent.GetAuthorizedKey()
	if err != nil {
		le.Printf("%s\n", err)
		exit(1)
	}

	le.Printf("Your ssh pubkey (on stdout):\n")
	fmt.Fprintf(os.Stdout, "%s", authorizedKey)

	if !showPubkeyOnly {
		if err = agent.Serve(sockPath); err != nil {
			le.Printf("%s\n", err)
			exit(1)
		}
	}

	exit(0)
}

func listPorts() error {
	ports, err := serial.GetPortsList()
	if err != nil {
		return fmt.Errorf("GetPortsList: %w", err)
	}
	for _, port := range ports {
		fmt.Printf("%s\n", port)
	}
	return nil
}
