package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh"
)

// Use when printing err/diag msgs
var le = log.New(os.Stderr, "", 0)

func main() {
	syscall.Umask(0o077)

	var sockPath, devPath string
	var onlyKeyOutput bool
	flag.CommandLine.SetOutput(os.Stderr)
	flag.StringVar(&sockPath, "a", "", "Path to bind agent's UNIX domain socket at")
	flag.BoolVar(&onlyKeyOutput, "k", false, "Don't start the agent, just output the ssh-ed25519 pubkey")
	flag.StringVar(&devPath, "port", "/dev/ttyACM0", "Path to serial port device")
	flag.Parse()

	if onlyKeyOutput && sockPath != "" {
		le.Printf("Can't combine -a and -k.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	if !onlyKeyOutput && sockPath == "" {
		le.Printf("Please pass at least -a or -k.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	le.Printf("Using serial port at %v\n", devPath)

	_, err := os.Stat(sockPath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		le.Printf("%s exists?\n", sockPath)
		os.Exit(1)
	}

	signer, err := NewMKDFSigner(devPath)
	if err != nil {
		if errors.Is(err, ErrMaybeWrongDevice) {
			le.Printf("If the serial port is correct for the device, then it might not be it\n" +
				"firmware-mode. Please unplug and plug it in again.\n")
		} else {
			le.Printf("%s\n", err)
		}
		os.Exit(1)
	}
	exit := func(code int) {
		if err := signer.disconnect(); err != nil {
			le.Printf("%s\n", err)
		}
		os.Exit(code)
	}

	agent, err := NewSSHAgent(signer)
	if err != nil {
		le.Printf("%s\n", err)
		exit(1)
	}

	sshPub, err := agent.GetSSHPub()
	if err != nil {
		le.Printf("%s\n", err)
		exit(1)
	}

	authorizedKey := ssh.MarshalAuthorizedKey(sshPub)
	le.Printf("Your ssh pubkey (on stdout):\n")
	fmt.Fprintf(os.Stdout, "%s", authorizedKey)
	if onlyKeyOutput {
		exit(0)
	}

	err = agent.Serve(sockPath)
	if err != nil {
		le.Printf("%s\n", err)
		exit(1)
	}

	exit(0)
}
