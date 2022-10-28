// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/tk1"
	"go.bug.st/serial/enumerator"
)

const (
	tillitisUSBVID = "1207"
	tillitisUSBPID = "8887"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

func main() {
	syscall.Umask(0o077)

	exit := func(code int) {
		os.Exit(code)
	}

	var sockPath, devPath, fileUSS string
	var speed int
	var enterUSS, showPubkeyOnly, listPortsOnly bool
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&sockPath, "agent-socket", "a", "",
		"Start the agent, setting the `PATH` to the UNIX-domain socket that it should bind to. SSH finds and talks to the agent if given this path in the environment variable SSH_AUTH_SOCK.")
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "k", false,
		"Don't start the agent, only output the ssh-ed25519 public key.")
	pflag.BoolVarP(&listPortsOnly, "list-ports", "", false,
		"List possible serial ports to use with --port.")
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", tk1.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&enterUSS, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto the USB stick along with the app itself. A different USS results in different SSH public/private keys, meaning a different identity.")
	pflag.StringVar(&fileUSS, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n%s", os.Args[0],
			pflag.CommandLine.FlagUsagesWrapped(80))
	}
	pflag.Parse()

	if pflag.NArg() > 0 {
		le.Printf("Unexpected argument: %s\n\n", strings.Join(pflag.Args(), " "))
		pflag.Usage()
		exit(2)
	}

	exclusive := 0
	if sockPath != "" {
		exclusive++
	}
	if showPubkeyOnly {
		exclusive++
	}
	if listPortsOnly {
		exclusive++
	}
	if exclusive > 1 {
		le.Printf("Pass only one of -a, -k, or --list-ports.\n\n")
		pflag.Usage()
		exit(2)
	}

	if listPortsOnly {
		n, err := printPorts()
		if err != nil {
			le.Printf("Failed to list ports: %v\n", err)
			exit(1)
		} else if n == 0 {
			exit(1)
		}
		// Successful only if we found some port
		exit(0)
	}

	if !showPubkeyOnly && sockPath == "" {
		le.Printf("Please pass at least -a or -k.\n\n")
		pflag.Usage()
		exit(2)
	}

	if enterUSS && fileUSS != "" {
		le.Printf("Pass only one of --uss or --uss-file.\n\n")
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

	if devPath == "" {
		var err error
		devPath, err = detectPort()
		if err != nil {
			le.Printf("Failed to list ports: %v\n", err)
			exit(1)
		} else if devPath == "" {
			exit(1)
		}
	}

	signer, err := NewSigner(devPath, speed, enterUSS, fileUSS)
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

type serialPort struct {
	devPath      string
	serialNumber string
}

func detectPort() (string, error) {
	ports, err := getTillitisPorts()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		le.Printf("Could not detect any Tillitis Key serial ports.\n" +
			"You may still use the --port flag to use a known device path.\n")
		return "", nil
	}
	if len(ports) > 1 {
		le.Printf("Detected %d Tillitis Key serial ports:\n", len(ports))
		for _, p := range ports {
			le.Printf("%s with serial number %s\n", p.devPath, p.serialNumber)
		}
		le.Printf("Please choose one of the above by using the --port flag.\n")
		return "", nil
	}
	le.Printf("Auto-detected serial port %s\n", ports[0].devPath)
	return ports[0].devPath, nil
}

func printPorts() (int, error) {
	ports, err := getTillitisPorts()
	if err != nil {
		return 0, err
	}
	if len(ports) == 0 {
		le.Printf("No Tillitis Key serial ports found.\n")
	} else {
		le.Printf("Tillitis Key serial ports (on stdout):\n")
		for _, p := range ports {
			fmt.Fprintf(os.Stdout, "%s serialNumber:%s\n", p.devPath, p.serialNumber)
		}
	}
	return len(ports), nil
}

func getTillitisPorts() ([]serialPort, error) {
	var ports []serialPort
	portDetails, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("GetDetailedPortsList: %w", err)
	}
	if len(portDetails) == 0 {
		return ports, nil
	}
	for _, port := range portDetails {
		if port.IsUSB && port.VID == tillitisUSBVID && port.PID == tillitisUSBPID {
			ports = append(ports, serialPort{port.Name, port.SerialNumber})
		}
	}
	return ports, nil
}
