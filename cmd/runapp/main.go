package main

import (
	"crypto/ed25519"
	"fmt"
	"os"

	"github.com/tillitis/tillitis-key1-apps/mkdf"
	"github.com/tillitis/tillitis-key1-apps/mkdfsign"
	"github.com/spf13/pflag"
	"go.bug.st/serial"
)

func main() {
	fileName := pflag.String("file", "", "Name of file to be uploaded")
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", 38400, "When talking over the serial port, bits per second")
	pflag.Parse()
	// mkdf.SilenceLogging()

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	conn, err := serial.Open(*port, &serial.Mode{BaudRate: *speed})
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}
	exit := func(code int) {
		conn.Close()
		os.Exit(code)
	}

	if *fileName != "" {
		nameVer, err := mkdf.GetNameVersion(conn)
		if err != nil {
			fmt.Printf("GetNameVersion failed: %v\n", err)
			fmt.Printf("If the serial port device is correct, then the device might not be in\n" +
				"firmware-mode (and already have an app running). Please unplug and plug it in again.\n")
			exit(1)
		}
		fmt.Printf("Firmware has name0:%s name1:%s version:%d\n",
			nameVer.Name0, nameVer.Name1, nameVer.Version)
		fmt.Printf("Loading app onto device\n")
		err = mkdf.LoadAppFromFile(conn, *fileName)
		if err != nil {
			fmt.Printf("LoadAppFromFile failed: %v\n", err)
			exit(1)
		}
	} else {
		fmt.Printf("No app filename given, assuming app is already running\n")
	}

	pubkey, err := mkdfsign.GetPubkey(conn)
	if err != nil {
		fmt.Printf("GetPubKey failed: %v\n", err)
		exit(1)
	}
	fmt.Printf("Public Key from device: %x\n", pubkey)

	var message []byte
	for i := 0; i < 4096; i++ {
		message = append(message, byte(i))
	}

	fmt.Printf("Message size: %v, message: %x\n", len(message), message)
	signature, err := mkdfsign.Sign(conn, message)
	if err != nil {
		fmt.Printf("Sign failed: %v\n", err)
		exit(1)
	}
	fmt.Printf("Signature over message by device: %x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		fmt.Printf("Signature did NOT verify.\n")
	} else {
		fmt.Printf("Signature verified.\n")
	}

	exit(0)
}
