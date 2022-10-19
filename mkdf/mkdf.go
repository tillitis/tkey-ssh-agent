// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// Package mkdf provides a connection to a Tillitis Key 1 security stick.
// To create a new connection:
//
//	tk, err := mkdf.New(*port, *speed)
//
// Then you can start using it by asking it to identify itself:
//
//	nameVer, err := tk.GetNameVersion()
//
// Or uploading and starting an app on the stick:
//
//	err = tk.LoadAppFromFile(*fileName)
//
// After this, you will have to switch to a new protocol specific to
// the app, see for instance the package
// github.com/tillitis/tillitis-key1-apps/mkdfsign for one such app
// specific protocol.
//
// When writing your app specific protocol you might still want to use
// the framing protocol provided here. See NewFrameBuf() and
// ReadFrame().
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
	// Speed in bps for talking to Tillitis Key 1
	SerialSpeed = 62500
	// Codes used in app proto responses
	StatusOK  = 0x00
	StatusBad = 0x01
)

// TillitisKey is a serial connection to a Tillitis Key 1 and the
// commands that the firmware supports.
type TillitisKey struct {
	conn serial.Port
}

// New() opens a connection to the Tillitis Key 1 at the serial device
// port at indicated speed.
func New(port string, speed int) (TillitisKey, error) {
	var tk TillitisKey
	var err error

	tk.conn, err = serial.Open(port, &serial.Mode{BaudRate: speed})
	if err != nil {
		return tk, fmt.Errorf("Open %s: %w", port, err)
	}

	return tk, nil
}

// Close the connection to the TK1
func (tk TillitisKey) Close() error {
	if err := tk.conn.Close(); err != nil {
		return fmt.Errorf("conn.Close: %w", err)
	}
	return nil
}

// SetReadTimeout sets the timeout of the underlying serial connection to the
// TK1. Pass 0 seconds to not have any timeout. Note that the timeout
// implemented in the serial lib only works for simple Read(). E.g.
// io.ReadFull() will Read() until the buffer is full.
func (tk TillitisKey) SetReadTimeout(seconds int) error {
	var t time.Duration = -1
	if seconds > 0 {
		t = time.Duration(seconds) * time.Second
	}
	if err := tk.conn.SetReadTimeout(t); err != nil {
		return fmt.Errorf("SetReadTimeout: %w", err)
	}
	return nil
}

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

// GetNameVersion() gets the name and version from the TK1 firmware
func (tk TillitisKey) GetNameVersion() (*NameVersion, error) {
	id := 2
	tx, err := NewFrameBuf(cmdGetNameVersion, id)
	if err != nil {
		return nil, err
	}

	Dump("GetNameVersion tx", tx)
	if err = tk.Write(tx); err != nil {
		return nil, err
	}

	if err = tk.SetReadTimeout(2); err != nil {
		return nil, err
	}

	rx, _, err := tk.ReadFrame(rspGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if err = tk.SetReadTimeout(0); err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	nameVer := &NameVersion{}
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

// LoadAppFromFile() loads and runs a raw binary file from fileName into the TK1.
func (tk TillitisKey) LoadAppFromFile(fileName string, secretPhrase []byte) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}

	return tk.LoadApp(content, secretPhrase)
}

// LoadApp loads the USS (User Supplied Secret), and contents of bin
// into the TK1, running the app after verifying that the digest
// calculated on the host is the same as the digest from the TK1.
//
// The USS is a 32 bytes digest hashed from secretPhrase (which is
// provided by the user). If secretPhrase is an empty slice, 32 bytes
// of zeroes will be loaded as USS.
//
// Loading USS is always done together with loading and running an
// app, because the host program can't otherwise be sure that the
// expected USS is used.
func (tk TillitisKey) LoadApp(bin []byte, secretPhrase []byte) error {
	binLen := len(bin)
	if binLen > 65536 {
		return fmt.Errorf("File to big")
	}

	err := tk.loadUSS(secretPhrase)
	if err != nil {
		return err
	}

	le.Printf("app size: %v, 0x%x, 0b%b\n", binLen, binLen, binLen)

	err = tk.setAppSize(binLen)
	if err != nil {
		return err
	}

	// Load the file
	var offset int
	for nsent := 0; offset < binLen; offset += nsent {
		nsent, err = tk.loadAppData(bin[offset:])
		if err != nil {
			return fmt.Errorf("loadAppData: %w", err)
		}
	}
	if offset > binLen {
		return fmt.Errorf("transmitted more than expected")
	}

	le.Printf("Going to getappdigest\n")
	appDigest, err := tk.getAppDigest()
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
	return tk.runApp()
}

func (tk TillitisKey) loadUSS(secretPhrase []byte) error {
	id := 2
	tx, err := NewFrameBuf(cmdLoadUSS, id)
	if err != nil {
		return err
	}

	if len(secretPhrase) == 0 {
		uss := [32]byte{}
		copy(tx[2:], uss[:])
	} else {
		// Hash user's phrase as USS
		uss := blake2s.Sum256(secretPhrase)
		copy(tx[2:], uss[:])
	}

	// Not running Dump() on the secret USS
	le.Printf("LoadUSS tx len:%d contents omitted\n", len(tx))
	if err = tk.Write(tx); err != nil {
		return err
	}

	rx, _, err := tk.ReadFrame(rspLoadUSS, id)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return fmt.Errorf("LoadUSS NOK")
	}

	return nil
}

// setAppSize() sets the size of the app to be loaded into the TK1.
func (tk TillitisKey) setAppSize(size int) error {
	id := 2
	tx, err := NewFrameBuf(cmdLoadAppSize, id)
	if err != nil {
		return err
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)

	Dump("SetAppSize tx", tx)
	if err = tk.Write(tx); err != nil {
		return err
	}

	rx, _, err := tk.ReadFrame(rspLoadAppSize, id)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return fmt.Errorf("SetAppSize NOK")
	}

	return nil
}

// loadAppData() loads a chunk of the raw app binary into the TK1 and
// waits for a reply.
func (tk TillitisKey) loadAppData(content []byte) (int, error) {
	id := 2
	tx, err := NewFrameBuf(cmdLoadAppData, id)
	if err != nil {
		return 0, err
	}

	payload := make([]byte, CmdLen128.Bytelen()-1)
	copied := copy(payload, content)

	// Add padding if not filling the payload buffer.
	if copied < len(payload) {
		padding := make([]byte, len(payload)-copied)
		copy(payload[copied:], padding)
	}

	copy(tx[2:], payload)

	Dump("LoadAppData tx", tx)

	if err = tk.Write(tx); err != nil {
		return 0, err
	}

	// Wait for reply
	rx, _, err := tk.ReadFrame(rspLoadAppData, id)
	if err != nil {
		return 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return 0, fmt.Errorf("LoadAppData NOK")
	}

	return copied, nil
}

// getAppDigest() asks for an app digest from the TK1.
func (tk TillitisKey) getAppDigest() ([32]byte, error) {
	var md [32]byte
	id := 2
	tx, err := NewFrameBuf(cmdGetAppDigest, id)
	if err != nil {
		return md, err
	}

	Dump("GetDigest tx", tx)

	if err = tk.Write(tx); err != nil {
		return md, err
	}

	// Wait for reply
	rx, _, err := tk.ReadFrame(rspGetAppDigest, id)
	if err != nil {
		return md, fmt.Errorf("ReadFrame: %w", err)
	}

	copy(md[:], rx[2:])

	return md, nil
}

// runApp() runs the loaded app, if any, in the TK1.
func (tk TillitisKey) runApp() error {
	id := 2
	tx, err := NewFrameBuf(cmdRunApp, id)
	if err != nil {
		return err
	}

	if err = tk.Write(tx); err != nil {
		return err
	}

	// Wait for reply
	rx, _, err := tk.ReadFrame(rspRunApp, id)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
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
