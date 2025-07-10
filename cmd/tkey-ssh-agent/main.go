// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

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

	"github.com/spf13/pflag"
	"github.com/tillitis/tkeyclient"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

const progname = "tkey-ssh-agent"

var version string

const windowsPipePrefix = `\\.\pipe\`

type Port struct {
	Path  string
	Speed int
}

type UssConfig struct {
	EnterManually bool
	Path          string
	PinentryPath  string
}

func main() {
	exit := func(code int) {
		os.Exit(code)
	}

	if version == "" {
		version = readBuildInfo()
	}

	var port Port
	var ussConf UssConfig
	var agentPath string
	var showPubkeyOnly, listPortsOnly, versionOnly, helpOnly bool
	pflag.CommandLine.SetOutput(os.Stderr)
	pflag.CommandLine.SortFlags = false
	pflag.CommandLine.SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		// Make old name agent-socket an alias for the new name
		if name == "agent-socket" {
			name = "agent-path"
		}
		return pflag.NormalizedName(name)
	})
	pflag.StringVarP(&agentPath, "agent-path", "a", "",
		fmt.Sprintf("Start the agent, setting the `PATH` to the UNIX-domain socket that it should listen on. On Windows, a Named Pipe at '%s\\PATH' will be used.", windowsPipePrefix))
	pflag.BoolVarP(&showPubkeyOnly, "show-pubkey", "p", false,
		"Don't start the agent, only output the ssh-ed25519 public key.")
	pflag.BoolVarP(&listPortsOnly, "list-ports", "L", false,
		"List possible serial ports to use with --port.")
	pflag.StringVar(&port.Path, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&port.Speed, "speed", tkeyclient.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&ussConf.EnterManually, "uss", false,
		"Enable typing of a phrase to be hashed as the User Supplied Secret. The USS is loaded onto the TKey along with the app itself. A different USS results in different SSH public/private keys, meaning a different identity.")
	pflag.StringVar(&ussConf.Path, "uss-file", "",
		"Read `FILE` and hash its contents as the USS. Use '-' (dash) to read from stdin. The full contents are hashed unmodified (e.g. newlines are not stripped).")
	pflag.StringVar(&ussConf.PinentryPath, "pinentry", "",
		"Pinentry `PROGRAM` for use by --uss. The default is found by looking in your gpg-agent.conf for pinentry-program, or 'pinentry' if not found there. On Windows, an attempt is made to find Gpg4win's pinentry program to use as default. On macOS, a native prompt is used by default.")
	pflag.BoolVar(&versionOnly, "version", false, "Output version information.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	pflag.Usage = func() {
		desc := fmt.Sprintf(`Usage: %[1]s -a|-p|-L [flags...]

%[1]s is an alternative SSH agent that communicates with a Tillitis TKey
USB stick. This stick holds private key and signing functionality for public key
authentication.

When the environment variable SSH_AUTH_SOCK is set to contain the agent-path,
programs like ssh(1) and ssh-keygen(1) can find and use this agent, e.g. for
authentication when accessing other machines.

To make the TKey provide this functionality, the %[1]s contains a compiled
signer app binary which it loads onto the stick and starts. The LED on the stick
is a steady blue when the signer app is running and waiting for a command. It
will flash green when the stick must be touched to complete a signature.`, progname)
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
		fmt.Printf("%s %s\n\n", progname, version)
		fmt.Printf("Embedded device apps:\n")
		for _, app := range ListApps() {
			fmt.Printf("%s\nSHA512: %s\n", app.name, app.digest)
		}
		exit(0)
	}

	exclusive := 0
	if agentPath != "" {
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

	if !showPubkeyOnly && agentPath == "" {
		le.Printf("Please pass at least -a or -p.\n\n")
		pflag.Usage()
		exit(2)
	}

	if ussConf.EnterManually && ussConf.Path != "" {
		le.Printf("Pass only one of --uss or --uss-file.\n\n")
		pflag.Usage()
		exit(2)
	}

	prevExitFunc := exit
	exit = func(code int) {
		_ = os.Remove(agentPath)
		prevExitFunc(code)
	}

	signer := NewSigner(port, ussConf, exit)

	if showPubkeyOnly {
		if !signer.connect() {
			le.Printf("Connect failed")
			prevExitFunc(1)
		}
		signer.printAuthorizedKey()
		signer.closeNow()
		prevExitFunc(0)
	}

	if runtime.GOOS == "windows" {
		agentPath = filepath.Join(windowsPipePrefix, agentPath)
	} else {
		var err error
		agentPath, err = filepath.Abs(agentPath)
		if err != nil {
			le.Printf("Failed to resolve socket path: %s", err)
			prevExitFunc(1)
		}
	}

	_, err := os.Stat(agentPath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("Is an agent already running? Path %s exists.", agentPath)
		notify(msg)
		le.Printf("%s\n", msg)
		// Don't remove the socket for the agent running.
		prevExitFunc(1)
	}

	agent := NewSSHAgent(signer)
	if err := agent.Serve(agentPath); err != nil {
		le.Printf("%s\n", err)
		exit(1)
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
	ports, err := tkeyclient.GetSerialPorts()
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
