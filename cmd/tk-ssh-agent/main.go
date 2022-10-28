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
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tillitis-key1-apps/tk1"
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
		"Start the agent, setting the `PATH` to the UNIX-domain socket that it should bind/listen to.")
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "k", false,
		"Don't start the agent, only output the ssh-ed25519 public key.")
	pflag.BoolVarP(&listPortsOnly, "list-ports", "L", false,
		"List possible serial ports to use with --port.")
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", tk1.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&enterUSS, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto Tillitis Key along with the app itself. A different USS results in different SSH public/private keys, meaning a different identity.")
	pflag.StringVar(&fileUSS, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")
	pflag.Usage = func() {
		desc := `Usage: tk-ssh-agent -a|-k|-L [flags...]

tk-ssh-agent is an alternative ssh-agent that communicates with a Tillitis Key 1
USB stick. This stick holds private key and signing functionality for public key
authentication.

Through the agent-socket, when set in the SSH_AUTH_SOCK environment variable,
programs like ssh(1) and ssh-keygen(1) can find and use this agent, e.g. for
authentication when accessing other machines.

To make the Tillitis Key 1 provide this functionality, the tk-ssh-agent contains
a compiled signerapp binary which it loads onto the stick and starts. The stick
will flash blue when signerapp is running and waiting for a signing command, and
green when the stick must be touched to complete a signature.`
		fmt.Fprintf(os.Stderr, "%s\n\n%s", desc,
			pflag.CommandLine.FlagUsagesWrapped(86))
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
		le.Printf("Pass only one of -a, -k, or -L.\n\n")
		pflag.Usage()
		exit(2)
	}

	if listPortsOnly {
		n, err := printPorts()
		if err != nil {
			le.Printf("%v\n", err)
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
		devPath, err = util.DetectSerialPort()
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
			le.Printf("If the serial port is correct, then Tillitis Key might not be in firmware-\n" +
				"mode, and have an app running already. Please unplug and plug it in again.\n")
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

func printPorts() (int, error) {
	ports, err := util.GetSerialPorts()
	if err != nil {
		return 0, fmt.Errorf("Failed to list ports: %w", err)
	}
	if len(ports) == 0 {
		le.Printf("No Tillitis Key serial ports found.\n")
	} else {
		le.Printf("Tillitis Key serial ports (on stdout):\n")
		for _, p := range ports {
			fmt.Fprintf(os.Stdout, "%s serialNumber:%s\n", p.DevPath, p.SerialNumber)
		}
	}
	return len(ports), nil
}
