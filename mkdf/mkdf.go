// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package mkdf

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.bug.st/serial"
	"golang.org/x/crypto/blake2s"
)

var le = log.New(os.Stderr, "", 0)

func SilenceLogging() {
	le.SetOutput(io.Discard)
}

const (
	StatusOK  = 0x00
	StatusBad = 0x01
)

type NameVersion struct {
	Name0   string
	Name1   string
	Version uint32
}

func (n *NameVersion) Unpack(raw []byte) {
	n.Name0 = fmt.Sprintf("%c%c%c%c", raw[3], raw[2], raw[1], raw[0])
	n.Name1 = fmt.Sprintf("%c%c%c%c", raw[7], raw[6], raw[5], raw[4])
	n.Version = binary.LittleEndian.Uint32(raw[8:12])
}

func GetNameVersion(conn serial.Port) (*NameVersion, error) {
	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		CmdLen:   CmdLen1,
	}

	// This sets 2s timeout, see: https://github.com/bugst/go-serial/issues/141
	err := conn.SetReadTimeout(2_000 / 100 * time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	tx, err := packSimple(hdr, fwCmdGetNameVersion)
	if err != nil {
		return nil, fmt.Errorf("packSimple: %w", err)
	}

	Dump("GetNameVersion tx", tx)
	if err = Xmit(conn, tx); err != nil {
		return nil, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(conn, fwRspGetNameVersion, hdr.ID, CmdLen32)
	if err != nil {
		return nil, fmt.Errorf("fwRecv: %w", err)
	}

	err = conn.SetReadTimeout(serial.NoTimeout)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	nameVer := &NameVersion{}
	nameVer.Unpack(rx)

	return nameVer, nil
}

func LoadAppFromFile(conn serial.Port, fileName string) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}
	return LoadApp(conn, content)
}

func LoadApp(conn serial.Port, bin []byte) error {
	binLen := len(bin)
	if binLen > 65536 {
		return fmt.Errorf("File to big")
	}

	le.Printf("app size: %v, 0x%x, 0b%b\n", binLen, binLen, binLen)

	err := setAppSize(conn, binLen)
	if err != nil {
		return err
	}

	// Load the file
	var offset int
	for nsent := 0; offset < binLen; offset += nsent {
		nsent, err = loadAppData(conn, bin[offset:])
		if err != nil {
			return fmt.Errorf("loadAppData: %w", err)
		}
	}
	if offset > binLen {
		return fmt.Errorf("transmitted more than expected")
	}

	le.Printf("Going to getappdigest\n")
	appDigest, err := getAppDigest(conn)
	if err != nil {
		return err
	}

	digest := blake2s.Sum256(bin)

	le.Printf("Digest from host:\n")
	printDigest(digest)
	le.Printf("Digest from device:\n")
	printDigest(appDigest)

	if appDigest != digest {
		return fmt.Errorf("Different digests")
	}
	le.Printf("Same digests!\n")

	// Run the app
	le.Printf("Running the app\n")
	return runApp(conn)
}

func setAppSize(conn serial.Port, size int) error {
	appsize := appSize{
		hdr: Frame{
			ID:       2,
			Endpoint: DestFW,
			CmdLen:   CmdLen32,
		},
		size: size,
	}

	tx, err := appsize.pack()
	if err != nil {
		return err
	}

	Dump("SetAppSize tx", tx)
	if err = Xmit(conn, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(conn, fwRspLoadAppSize, appsize.hdr.ID, CmdLen4)
	if err != nil {
		return fmt.Errorf("fwRecv: %w", err)
	}
	if rx[0] != StatusOK {
		return fmt.Errorf("SetAppSize NOK")
	}

	return nil
}

func loadAppData(conn serial.Port, content []byte) (int, error) {
	cmdLen := CmdLen128
	appdata := appData{
		hdr: Frame{
			ID:       2,
			Endpoint: DestFW,
			CmdLen:   cmdLen,
		},
		// Payload len is cmdlen minus the fw cmd byte
		data: make([]byte, cmdLen.Bytelen()-1),
	}

	nsent := appdata.copy(content)

	tx, err := appdata.pack()
	if err != nil {
		return 0, err
	}

	Dump("LoadAppData tx", tx)
	if err = Xmit(conn, tx); err != nil {
		return 0, fmt.Errorf("Xmit: %w", err)
	}

	// Wait for reply
	rx, err := fwRecv(conn, fwRspLoadAppData, appdata.hdr.ID, CmdLen4)
	if err != nil {
		return 0, err
	}

	if rx[0] != StatusOK {
		return 0, fmt.Errorf("LoadAppData NOK")
	}

	return nsent, nil
}

func getAppDigest(conn serial.Port) ([32]byte, error) {
	var md [32]byte

	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		CmdLen:   CmdLen1,
	}

	// Check the digest
	tx, err := packSimple(hdr, fwCmdGetAppDigest)
	if err != nil {
		return md, fmt.Errorf("packSimple: %w", err)
	}

	Dump("GetDigest tx", tx)
	if err = Xmit(conn, tx); err != nil {
		return md, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(conn, fwRspGetAppDigest, hdr.ID, CmdLen128)
	if err != nil {
		return md, err
	}

	copy(md[:], rx)

	return md, nil
}

func runApp(conn serial.Port) error {
	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		CmdLen:   CmdLen1,
	}

	tx, err := packSimple(hdr, fwCmdRunApp)
	if err != nil {
		return fmt.Errorf("packSimple: %w", err)
	}

	Dump("RunApp tx", tx)
	if err = Xmit(conn, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(conn, fwRspRunApp, hdr.ID, CmdLen4)
	if err != nil {
		return err
	}

	if rx[0] != StatusOK {
		return fmt.Errorf("RunApp NOK")
	}

	return nil
}

func printDigest(md [32]byte) {
	digest := ""
	for j := 0; j < 4; j++ {
		for i := 0; i < 8; i++ {
			digest += fmt.Sprintf("%02x", md[i+8*j])
		}
		digest += " "
	}
	le.Printf(digest + "\n")
}
