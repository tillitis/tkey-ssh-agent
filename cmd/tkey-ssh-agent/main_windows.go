// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

//go:build windows

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/kardianos/service"
	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tillitis-key1-apps/tk1"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

const progname = "tkey-ssh-agent"

var version string

type program struct{}

func (p *program) Start(s service.Service) error {
    go p.run()
    return nil
}

func (p *program) run() {
	runAgent()
}

func (p *program) Stop(s service.Service) error {
    // Stop your service code here
    return nil
}

func main() {
	// If no pflags are being used, start as a windows service
	// Running the application directly as a service introduces exiting issues
	if len(os.Args) > 1{
	svcConfig := &service.Config{
        Name:        "tkey ssh agent",
        DisplayName: "TKey SSH Agent",
        Description: "Tkey SSH Agent is an alternative SSH agent that communicates with a Tillitis TKey",
		Arguments: []string{"-a agent.sock"},
    }

    prg := &program{}

    s, err := service.New(prg, svcConfig)
    if err != nil {
        log.Fatal(err)
    }
        arg := os.Args[1]
        if arg == "install" {
            err = s.Install()
            if err != nil {
                log.Fatal(err)
            }
			s.Start()
            fmt.Println("Service installed and started successfully.")
            return
        } else if arg == "uninstall" {
            err = s.Uninstall()
            if err != nil {
                log.Fatal(err)
            }
			s.Stop()
            fmt.Println("Service stopped and uninstalled successfully.")
            return
        }
    

    err = s.Run()
    if err != nil {
        log.Fatal(err)
    }
	} else {
		//Running service-less
		runAgent()
	}
}

func runAgent() {
	// syscall.Umask(0o077)

	exit := func(code int) {
		os.Exit(code)
	}

	if version == "" {
		version = readBuildInfo()
	}

	var sockPath, devPath, fileUSS, pinentry string
	var speed int
	var enterUSS, showPubkeyOnly, listPortsOnly, versionOnly, helpOnly bool
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&sockPath, "agent-socket", "a", "",
		"Start the agent, setting the `PATH` to the UNIX-domain socket that it should bind/listen to.")
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "p", false,
		"Don't start the agent, only output the ssh-ed25519 public key.")
	pflag.BoolVarP(&listPortsOnly, "list-ports", "L", false,
		"List possible serial ports to use with --port.")
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", tk1.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&enterUSS, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto the TKey along with the app itself. A different USS results in different SSH public/private keys, meaning a different identity.")
	pflag.StringVar(&fileUSS, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")
	pflag.StringVar(&pinentry, "pinentry", "",
		"Pinentry `PROGRAM` for use by --uss. The default is found by looking in your gpg-agent.conf for pinentry-program, or 'pinentry' if not found there.")
	pflag.BoolVar(&versionOnly, "version", false, "Output version information.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	pflag.Usage = func() {
		desc := fmt.Sprintf(`Usage: %[1]s -a|-p|-L [flags...]

%[1]s is an alternative SSH agent that communicates with a Tillitis TKey
USB stick. This stick holds private key and signing functionality for public key
authentication.

Through the agent-socket, when set in the SSH_AUTH_SOCK environment variable,
programs like ssh(1) and ssh-keygen(1) can find and use this agent, e.g. for
authentication when accessing other machines.

To make the TKey provide this functionality, the %[1]s contains a compiled
signer app binary which it loads onto the stick and starts. The stick will flash
blue when the signer app is running and waiting for a signing command, and
green when the stick must be touched to complete a signature.`, progname)
		le.Printf("%s\n\n%s", desc,
			pflag.CommandLine.FlagUsagesWrapped(86))
	}
	pflag.Parse()

	if pflag.NArg() > 0 {
		le.Printf("Unexpected argument: %s\n\n", strings.Join(pflag.Args(), " "))
		pflag.Usage()
		exit(2)
	}

	if signerAppNoTouch != "" {
		le.Printf("WARNING! This tkey-ssh-agent and signer app is built with the touch requirement removed\n")
	}
	if helpOnly {
		pflag.Usage()
		exit(0)
	}
	if versionOnly {
		fmt.Printf("%s %s\n", progname, version)
		exit(0)
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
		le.Printf("Pass only one of -a, -p, or -L.\n\n")
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
		le.Printf("Please pass at least -a or -p.\n\n")
		pflag.Usage()
		exit(2)
	}

	if enterUSS && fileUSS != "" {
		le.Printf("Pass only one of --uss or --uss-file.\n\n")
		pflag.Usage()
		exit(2)
	}

	if sockPath != "" {
		// TODO hardcoding for now, and user has to pass -a dummy. Not
		// sure how to deal with flag parsing on windows. Also, should
		// it be possible to configure name of named pipe? openssh
		// ssh-agent on windows listens on \\.\pipe\openssh-ssh-agent
		// and it's not configurable.
		//
		// Btw this is how to check if a flag that has a default value
		// was actually passed:
		// if pflag.CommandLine.Changed("agent-socket") {
		// 	le.Printf("PASSED\n")
		// }

		if runtime.GOOS == "windows" {
			sockPath = `\\.\pipe\tkey-ssh-agent`
		} else {
			var err error
			sockPath, err = filepath.Abs(sockPath)
			if err != nil {
				le.Printf("Failed to get agent-socket path: %s", err)
				exit(1)
			}
			_, err = os.Stat(sockPath)
			if err == nil || !errors.Is(err, os.ErrNotExist) {
				le.Printf("Socket path %s exists?\n", sockPath)
				exit(1)
			}
			prevExitFunc := exit
			exit = func(code int) {
				_ = os.Remove(sockPath)
				prevExitFunc(code)
			}
		}
	}

	signer := NewSigner(devPath, speed, enterUSS, fileUSS, pinentry, exit)

	if !showPubkeyOnly {
		agent := NewSSHAgent(signer)
		if err := agent.Serve(sockPath); err != nil {
			le.Printf("%s\n", err)
			exit(1)
		}
	} else {
		if !signer.connect() {
			le.Printf("Connect failed")
			exit(1)
		}
		signer.closeNow()
	}

	exit(0)
}

func readBuildInfo() string {
	version := "devel without BuildInfo"
	if info, ok := debug.ReadBuildInfo(); ok {
		sb := strings.Builder{}
		sb.WriteString("devel")
		for _, setting := range info.Settings {
			if strings.HasPrefix(setting.Key, "vcs") {
				sb.WriteString(fmt.Sprintf(" %s=%s", setting.Key, setting.Value))
			}
		}
		version = sb.String()
	}
	return version
}

func printPorts() (int, error) {
	ports, err := util.GetSerialPorts()
	if err != nil {
		return 0, fmt.Errorf("Failed to list ports: %w", err)
	}
	if len(ports) == 0 {
		le.Printf("No TKey serial ports found.\n")
	} else {
		le.Printf("TKey serial ports (on stdout):\n")
		for _, p := range ports {
			fmt.Fprintf(os.Stdout, "%s serialNumber:%s\n", p.DevPath, p.SerialNumber)
		}
	}
	return len(ports), nil
}
