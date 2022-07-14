package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"golang.org/x/crypto/blake2s"
)

func printdigest(md [32]byte) {
	for j := 0; j < 4; j++ {
		for i := 0; i < 8; i++ {
			fmt.Printf("0x%02x ", md[i+8*j])
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")

}

func loadapp(conn net.Conn, fileName string) error {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("couldn't load file %v", fileName)
	}

	contentlen := len(content)
	if contentlen > 65536 {
		return fmt.Errorf("File to big")
	}

	digest := blake2s.Sum256(content)

	fmt.Printf("app size: %v, 0x%x, 0b%b, digest: \n", contentlen, contentlen, contentlen)

	err = SetAppSize(conn, contentlen)
	if err != nil {
		return err
	}

	// Load the file
	for i := 0; i < contentlen; i += 63 {
		err = LoadAppData(conn, content[i:])
	}

	fmt.Printf("Going to getappdigest\n")
	md, err := GetAppDigest(conn)
	if err != nil {
		return err
	}

	fmt.Printf("Digest from host: \n")
	printdigest(digest)
	fmt.Printf("Digest from device: \n")
	printdigest(md)

	if md != digest {
		return fmt.Errorf("Different digests")
	} else {
		fmt.Printf("Same digests!\n")
	}

	// Run the app
	fmt.Printf("Running the app\n")
	RunApp(conn)

	return nil
}

func sign(conn net.Conn, data []byte) ([]byte, error) {
	err := SignSetSize(conn, len(data))
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(data); i += 63 {
		err = SignLoad(conn, data[i:])
		if err != nil {
			return nil, err
		}
	}

	signature, err := GetSig(conn)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func main() {
	fileName := flag.String("file", "", "Name of file to be uploaded")
	flag.Parse()

	conn, err := connect()
	if err != nil {
		fmt.Printf("Couldn't connect to device")
		os.Exit(1)
	}
	defer conn.Close()

	err = loadapp(conn, *fileName)
	if err != nil {
		fmt.Printf("Couldn't load file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Getting the public key\n")
	pubkey, err := GetPubkey(conn)
	if err != nil {
		fmt.Printf("Couldn't get pubkey: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Public key: %x\n", pubkey)

	message := []byte{0x01, 0x02, 0x03}
	signature, err := sign(conn, message)
	if err != nil {
		fmt.Printf("Couldn't sign: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("signature: %x\n", signature)

	if !ed25519.Verify(pubkey, message, signature) {
		fmt.Printf("Signature did NOT verify.\n")
	} else {
		fmt.Printf("Signature verified.\n")
	}
}
