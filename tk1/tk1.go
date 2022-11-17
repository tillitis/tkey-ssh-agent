// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// Package tk1 provides a connection to a Tillitis Key 1 security
// stick. To create a new connection:
//
//	tk := tk1.New()
//	err := tk.Connect(port)
//
// Then you can start using it by asking it to identify itself:
//
//	nameVer, err := tk.GetNameVersion()
//
// Or loading and starting an app on the stick:
//
//	err = tk.LoadAppFromFile(*fileName)
//
// After this, you will have to switch to a new protocol specific to
// the app, see for instance the package
// github.com/tillitis/tillitis-key1-apps/tk1sign for one such app
// specific protocol.
//
// When writing your app specific protocol you might still want to use
// the framing protocol provided here. See NewFrameBuf() and
// ReadFrame().
package tk1

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
	speed int
	conn  serial.Port
}

// New allocates a new TK1. Use the Connect() method to actually
// open a connection.
func New() *TillitisKey {
	tk := &TillitisKey{}
	return tk
}

func WithSpeed(speed int) func(*TillitisKey) {
	return func(tk *TillitisKey) {
		tk.speed = speed
	}
}

// Connect() connects to a TK1 serial port using the provided port
// device, and speed as specified in New().
func (tk *TillitisKey) Connect(port string, options ...func(*TillitisKey)) error {
	var err error

	tk.speed = SerialSpeed
	for _, opt := range options {
		opt(tk)
	}

	tk.conn, err = serial.Open(port, &serial.Mode{BaudRate: tk.speed})
	if err != nil {
		return fmt.Errorf("Open %s: %w", port, err)
	}

	return nil
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
	n.Name0 = fmt.Sprintf("%c%c%c%c", raw[0], raw[1], raw[2], raw[3])
	n.Name1 = fmt.Sprintf("%c%c%c%c", raw[4], raw[5], raw[6], raw[7])
	n.Version = binary.LittleEndian.Uint32(raw[8:12])
}

// GetNameVersion gets the name and version from the TK1 firmware
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

// Modelled after how tpt.py (in tillitis-key1 repo) generates the UDI
type UDI struct {
	Unnamed   uint8 // 4 bits, hardcoded to 0 by tpt.py
	VendorID  uint16
	ProductID uint8
	Revision  uint8 // 4 bits
	Serial    uint32
}

func (u *UDI) String() string {
	return fmt.Sprintf("%01x%04x:%02x:%01x:%08x",
		u.Unnamed, u.VendorID, u.ProductID, u.Revision, u.Serial)
}

// Unpack unpacks the UDI parts from the raw 8 bytes (2 * 32-bit
// words) sent on the wire.
func (u *UDI) Unpack(raw []byte) {
	vpr := binary.LittleEndian.Uint32(raw[0:4])
	u.Unnamed = uint8((vpr >> 28) & 0xf)
	u.VendorID = uint16((vpr >> 12) & 0xffff)
	u.ProductID = uint8((vpr >> 4) & 0xff)
	u.Revision = uint8(vpr & 0xf)
	u.Serial = binary.LittleEndian.Uint32(raw[4:8])
}

// GetUDI gets the UDI (Unique Device ID) from the TK1 firmware
func (tk TillitisKey) GetUDI() (*UDI, error) {
	id := 2
	tx, err := NewFrameBuf(cmdGetUDI, id)
	if err != nil {
		return nil, err
	}

	Dump("GetUDI tx", tx)
	if err = tk.Write(tx); err != nil {
		return nil, err
	}

	rx, _, err := tk.ReadFrame(rspGetUDI, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return nil, fmt.Errorf("GetUDI NOK")
	}

	udi := &UDI{}
	udi.Unpack(rx[3 : 3+8])

	return udi, nil
}

// LoadAppFromFile loads and runs a raw binary file from fileName into
// the TK1.
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
	if binLen > 100*1024 { // TK1_APP_MAX_SIZE
		return fmt.Errorf("File too big")
	}

	le.Printf("app size: %v, 0x%x, 0b%b\n", binLen, binLen, binLen)

	err := tk.loadApp(binLen, secretPhrase)
	if err != nil {
		return err
	}

	// Load the file
	var offset int
	var deviceDigest [32]byte

	for nsent := 0; offset < binLen; offset += nsent {
		if binLen-offset <= CmdLen128.Bytelen()-1 {
			deviceDigest, nsent, err = tk.loadAppData(bin[offset:], true)
		} else {
			_, nsent, err = tk.loadAppData(bin[offset:], false)
		}
		if err != nil {
			return fmt.Errorf("loadAppData: %w", err)
		}
	}
	if offset > binLen {
		return fmt.Errorf("transmitted more than expected")
	}

	digest := blake2s.Sum256(bin)

	le.Printf("Digest from host:\n")
	printDigest(digest)
	le.Printf("Digest from device:\n")
	printDigest(deviceDigest)

	if deviceDigest != digest {
		return fmt.Errorf("Different digests")
	}
	le.Printf("Same digests!\n")

	// The app has now started automatically.
	return nil
}

// loadApp sets the size and USS of the app to be loaded into the TK1.
func (tk TillitisKey) loadApp(size int, secretPhrase []byte) error {
	id := 2
	tx, err := NewFrameBuf(cmdLoadApp, id)
	if err != nil {
		return err
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)

	if len(secretPhrase) == 0 {
		tx[6] = 0
	} else {
		tx[6] = 1
		// Hash user's phrase as USS
		uss := blake2s.Sum256(secretPhrase)
		copy(tx[6:], uss[:])
	}

	Dump("LoadApp tx", tx)
	if err = tk.Write(tx); err != nil {
		return err
	}

	rx, _, err := tk.ReadFrame(rspLoadApp, id)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return fmt.Errorf("LoadApp NOK")
	}

	return nil
}

// loadAppData loads a chunk of the raw app binary into the TK1.
func (tk TillitisKey) loadAppData(content []byte, last bool) ([32]byte, int, error) {
	id := 2
	tx, err := NewFrameBuf(cmdLoadAppData, id)
	if err != nil {
		return [32]byte{}, 0, err
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
		return [32]byte{}, 0, err
	}

	var rx []byte
	var expectedResp Cmd

	if last {
		expectedResp = rspLoadAppDataReady
	} else {
		expectedResp = rspLoadAppData
	}

	// Wait for reply
	rx, _, err = tk.ReadFrame(expectedResp, id)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return [32]byte{}, 0, fmt.Errorf("LoadAppData NOK")
	}

	if last {
		var digest [32]byte
		copy(digest[:], rx[3:])
		return digest, copied, nil
	}

	return [32]byte{}, copied, nil
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
