package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
	"github.com/tarm/serial"
)

func main() {
	fileName := flag.String("file", "", "Name of file to be uploaded")
	port := flag.String("port", "/dev/ttyACM0", "Serial port path")
	flag.Parse()

	// mkdf.SilenceLogging()

	conn, err := serial.OpenPort(&serial.Config{
		Name:        *port,
		Baud:        1000000,
		ReadTimeout: 3 * time.Second,
	})
	if err != nil {
		fmt.Printf("Couldn't connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if *fileName != "" {
		nameVer, err := mkdf.GetNameVersion(conn)
		if err != nil {
			fmt.Printf("GetNameVersion failed: %v\n", err)
			fmt.Printf("If the serial port device is correct, then the device might not be in\n" +
				"firmware-mode. Please unplug and plug it in again.\n")
			os.Exit(1)
		}
		fmt.Printf("Firmware has name0:%s name1:%s version:%d\n",
			nameVer.Name0, nameVer.Name1, nameVer.Version)
		fmt.Printf("Loading app onto device\n")
		err = mkdf.LoadAppFromFile(conn, *fileName)
		if err != nil {
			fmt.Printf("LoadAppFromFile failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("No app filename given, assuming app is already running\n")
	}

	pubkey, err := mkdf.GetPubkey(conn)
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Public Key from device: %x\n", pubkey)

	var message []byte
	for i := 0; i < 4096; i++ {
		message = append(message, byte(i))
	}

	fmt.Printf("Message size: %v, message: %x\n", len(message), message)
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
