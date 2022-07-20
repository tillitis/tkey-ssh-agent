package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh"
)

func main() {
	syscall.Umask(0o077)

	sockPath := flag.String("a", "", "Path to bind agent's UNIX domain socket at")
	devPath := flag.String("port", "/dev/ttyACM0", "Path to serial port device")
	flag.Parse()

	if *sockPath == "" {
		fmt.Printf("Give me: -a /path/for/agent.sock\n")
		os.Exit(2)
	}

	fmt.Printf("Using serial port at %v\n", *devPath)

	_, err := os.Stat(*sockPath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("%s exists?\n", *sockPath)
		os.Exit(1)
	}

	signer, err := NewMKDFSigner(*devPath)
	if err != nil {
		if errors.Is(err, ErrMaybeWrongDevice) {
			fmt.Printf("If the serial port is correct for the device, then it might not be it\n" +
				"firmware-mode. Please unplug and plug it in again.\n")
		} else {
			fmt.Printf("%s\n", err)
		}
		os.Exit(1)
	}

	agent, err := NewSSHAgent(signer)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	sshPub, err := agent.GetSSHPub()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	authorizedKey := ssh.MarshalAuthorizedKey(sshPub)
	fmt.Printf("your ssh pubkey:\n%s", authorizedKey)

	// // append pubkey to authorized_keys, for local testing using something
	// // like: SSH_AUTH_SOCK=$(pwd)/agent.sock ssh -F /dev/null localhost
	// home, err := os.UserHomeDir()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// f, err := os.OpenFile(home+"/.ssh/authorized_keys", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o0644)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer f.Close()
	// if _, err := f.Write(authorizedKey); err != nil {
	// 	log.Fatal(err)
	// }

	err = agent.Serve(*sockPath)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
