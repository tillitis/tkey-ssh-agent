package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/mullvad/mta1signer/mta1"
)

func main() {
	fileName := flag.String("file", "", "Name of file to be uploaded")
	flag.Parse()

	conn, err := net.Dial("tcp", "localhost:4444")
	if err != nil {
		fmt.Printf("Couldn't connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = mta1.LoadApp(conn, *fileName)
	if err != nil {
		fmt.Printf("LoadApp failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Getting the public key\n")
	pubkey, err := mta1.GetPubkey(conn)
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Public key: %x\n", pubkey)

	message := []byte{0x01, 0x02, 0x03}
	signature, err := mta1.Sign(conn, message)
	if err != nil {
		fmt.Printf("Sign failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Signature: %x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		fmt.Printf("Signature did NOT verify.\n")
	} else {
		fmt.Printf("Signature verified.\n")
	}
}
