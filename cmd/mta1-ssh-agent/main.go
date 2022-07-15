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

func main() {
	syscall.Umask(0o077)

	sockPath := flag.String("a", "", "Path to bind agent's UNIX domain socket at")
	flag.Parse()
	if *sockPath == "" {
		fmt.Printf("Give me: -a /path/for/agent.sock\n")
		os.Exit(2)
	}

	_, err := os.Stat(*sockPath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("%s exists?\n", *sockPath)
		os.Exit(1)
	}

	// TODO assumes that the app is already running, and many other things...
	signer, err := NewMKDFSigner()
	if err != nil {
		log.Fatal(err)
	}

	agent, err := NewSSHAgent(signer)
	if err != nil {
		log.Fatal(err)
	}

	sshPub, err := agent.GetSSHPub()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
}
