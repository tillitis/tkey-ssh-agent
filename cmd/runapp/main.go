package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
	"github.com/tarm/serial"
)

func main() {
	fileName := flag.String("file", "", "Name of file to be uploaded")
	port := flag.String("port", "/dev/ttyACM0", "Serial port path")
	flag.Parse()

	// mkdf.SilenceLogging()

	conn, err := serial.OpenPort(&serial.Config{Name: *port, Baud: 1000000})
	if err != nil {
		fmt.Printf("Couldn't connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if *fileName != "" {
		fmt.Printf("Loading app onto device\n")
		err = mkdf.LoadApp(conn, *fileName)
		if err != nil {
			fmt.Printf("LoadApp failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Not loading app onto device, assuming it's running\n")
	}

	pubkey, err := mkdf.GetPubkey(conn)
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Public Key from device: %x\n", pubkey)

	message := []byte{0x01, 0x02, 0x03}
	fmt.Printf("Message: %+v\n", message)
	signature, err := mkdf.Sign(conn, message)
	if err != nil {
		fmt.Printf("Sign failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Signature over message by device: %x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		fmt.Printf("Signature did NOT verify.\n")
	} else {
		fmt.Printf("Signature verified.\n")
	}
}
