package mta1

import (
	"fmt"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/blake2s"
)

func LoadApp(conn net.Conn, fileName string) error {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	contentlen := len(content)
	if contentlen > 65536 {
		return fmt.Errorf("File to big")
	}

	digest := blake2s.Sum256(content)

	fmt.Printf("app size: %v, 0x%x, 0b%b, digest: \n", contentlen, contentlen, contentlen)

	err = setAppSize(conn, contentlen)
	if err != nil {
		return err
	}

	// Load the file
	for i := 0; i < contentlen; i += 63 {
		err = loadAppData(conn, content[i:])
		if err != nil {
			return err
		}
	}

	fmt.Printf("Going to getappdigest\n")
	md, err := getAppDigest(conn)
	if err != nil {
		return err
	}

	fmt.Printf("Digest from host: \n")
	printDigest(digest)
	fmt.Printf("Digest from device: \n")
	printDigest(md)

	if md != digest {
		return fmt.Errorf("Different digests")
	}
	fmt.Printf("Same digests!\n")

	// Run the app
	fmt.Printf("Running the app\n")
	return runApp(conn)
}

func setAppSize(c net.Conn, size int) error {
	appsize := appSize{
		hdr: frame{
			id:       2,
			endpoint: destFW,
			msgLen:   frameLen32,
		},
		size: size,
	}

	tx, err := appsize.pack()
	if err != nil {
		return err
	}

	dump("SetAppSize tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspLoadAppSize, appsize.hdr.id, frameLen4)
	if rx[2] != 0 {
		return fmt.Errorf("SetAppSize NOK")
	}
	if err != nil {
		return fmt.Errorf("fwRecv: %w", err)
	}

	return nil
}

func loadAppData(c net.Conn, content []byte) error {
	appdata := appData{
		hdr: frame{
			id:       2,
			endpoint: destFW,
			msgLen:   frameLen64,
		},
	}

	appdata.copy(content)
	tx, err := appdata.pack()
	if err != nil {
		return err
	}

	dump("LoadAppData tx:", tx)
	xmit(c, tx)

	// Wait for reply
	rx, err := fwRecv(c, fwRspLoadAppData, appdata.hdr.id, frameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("LoadAppData NOK")
	}

	return nil
}

func getAppDigest(c net.Conn) ([32]byte, error) {
	var md [32]byte

	hdr := frame{
		id:       2,
		endpoint: destFW,
		msgLen:   frameLen1,
	}

	// Check the digest
	tx, err := packSimple(hdr, fwCmdGetAppDigest)
	if err != nil {
		return md, fmt.Errorf("packing packet: %w", err)
	}

	dump("GetDigest tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspGetAppDigest, hdr.id, frameLen64)
	if err != nil {
		return md, err
	}

	copy(md[:], rx)

	return md, nil
}

func runApp(c net.Conn) error {
	hdr := frame{
		id:       2,
		endpoint: destFW,
		msgLen:   frameLen1,
	}

	// Check the digest
	tx, err := packSimple(hdr, fwCmdRunApp)
	if err != nil {
		return nil
	}

	dump("RunApp tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspRunApp, hdr.id, frameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("RunApp NOK")
	}

	return nil
}

func printDigest(md [32]byte) {
	for j := 0; j < 4; j++ {
		for i := 0; i < 8; i++ {
			fmt.Printf("0x%02x ", md[i+8*j])
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
}
