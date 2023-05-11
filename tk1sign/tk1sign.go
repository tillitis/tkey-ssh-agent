// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// Package tk1sign provides a connection to the ed25519 signer app
// running on the TKey. You're expected to pass an existing connection
// to it, so use it like this:
//
//	tk := tkeyclient.New()
//	err := tk.Connect(port)
//	signer := tk1sign.New(tk)
//
// Then use it like this to get the public key of the TKey:
//
//	pubkey, err := signer.GetPubkey()
//
// And like this to sign a message:
//
//	signature, err := signer.Sign(message)
package tk1sign

import (
	"fmt"

	"github.com/tillitis/tkeyclient"
)

var (
	cmdGetPubkey      = appCmd{0x01, "cmdGetPubkey", tkeyclient.CmdLen1}
	rspGetPubkey      = appCmd{0x02, "rspGetPubkey", tkeyclient.CmdLen128}
	cmdSetSize        = appCmd{0x03, "cmdSetSize", tkeyclient.CmdLen32}
	rspSetSize        = appCmd{0x04, "rspSetSize", tkeyclient.CmdLen4}
	cmdSignData       = appCmd{0x05, "cmdSignData", tkeyclient.CmdLen128}
	rspSignData       = appCmd{0x06, "rspSignData", tkeyclient.CmdLen4}
	cmdGetSig         = appCmd{0x07, "cmdGetSig", tkeyclient.CmdLen1}
	rspGetSig         = appCmd{0x08, "rspGetSig", tkeyclient.CmdLen128}
	cmdGetNameVersion = appCmd{0x09, "cmdGetNameVersion", tkeyclient.CmdLen1}
	rspGetNameVersion = appCmd{0x0a, "rspGetNameVersion", tkeyclient.CmdLen32}
)

const MaxSignSize = 4096

type appCmd struct {
	code   byte
	name   string
	cmdLen tkeyclient.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() tkeyclient.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() tkeyclient.Endpoint {
	return tkeyclient.DestApp
}

func (c appCmd) String() string {
	return c.name
}

type Signer struct {
	tk *tkeyclient.TillitisKey // A connection to a TKey
}

// New allocates a struct for communicating with the ed25519 signer
// app running on the TKey. You're expected to pass an existing
// connection to it, so use it like this:
//
//	tk := tkeyclient.New()
//	err := tk.Connect(port)
//	signer := tk1sign.New(tk)
func New(tk *tkeyclient.TillitisKey) Signer {
	var signer Signer

	signer.tk = tk

	return signer
}

// Close closes the connection to the TKey
func (s Signer) Close() error {
	if err := s.tk.Close(); err != nil {
		return fmt.Errorf("tk.Close: %w", err)
	}
	return nil
}

// GetAppNameVersion gets the name and version of the running app in
// the same style as the stick itself.
func (s Signer) GetAppNameVersion() (*tkeyclient.NameVersion, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tkeyclient.Dump("GetAppNameVersion tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	err = s.tk.SetReadTimeout(2)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	err = s.tk.SetReadTimeout(0)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	nameVer := &tkeyclient.NameVersion{}
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

// GetPubkey fetches the public key of the signer.
func (s Signer) GetPubkey() ([]byte, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetPubkey, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tkeyclient.Dump("GetPubkey tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetPubkey, id)
	tkeyclient.Dump("GetPubKey rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	// Skip frame header & app header, returning size of ed25519 pubkey
	return rx[2 : 2+32], nil
}

// Sign signs the message in data and returns an ed25519 signature.
func (s Signer) Sign(data []byte) ([]byte, error) {
	err := s.setSize(len(data))
	if err != nil {
		return nil, fmt.Errorf("setSize: %w", err)
	}

	var offset int
	for nsent := 0; offset < len(data); offset += nsent {
		nsent, err = s.signLoad(data[offset:])
		if err != nil {
			return nil, fmt.Errorf("signLoad: %w", err)
		}
	}
	if offset > len(data) {
		return nil, fmt.Errorf("transmitted more than expected")
	}

	signature, err := s.getSig()
	if err != nil {
		return nil, fmt.Errorf("getSig: %w", err)
	}

	return signature, nil
}

// SetSize sets the size of the data to sign.
func (s Signer) setSize(size int) error {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdSetSize, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)
	tkeyclient.Dump("SetAppSize tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspSetSize, id)
	tkeyclient.Dump("SetAppSize rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return fmt.Errorf("SetSignSize NOK")
	}

	return nil
}

// signload loads a chunk of a message to sign and waits for a
// response from the signer.
func (s Signer) signLoad(content []byte) (int, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdSignData, id)
	if err != nil {
		return 0, fmt.Errorf("NewFrameBuf: %w", err)
	}

	payload := make([]byte, cmdSignData.CmdLen().Bytelen()-1)
	copied := copy(payload, content)

	// Add padding if not filling the payload buffer.
	if copied < len(payload) {
		padding := make([]byte, len(payload)-copied)
		copy(payload[copied:], padding)
	}

	copy(tx[2:], payload)

	tkeyclient.Dump("LoadSignData tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return 0, fmt.Errorf("Write: %w", err)
	}

	// Wait for reply
	rx, _, err := s.tk.ReadFrame(rspSignData, id)
	if err != nil {
		return 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return 0, fmt.Errorf("SignData NOK")
	}

	return copied, nil
}

// getSig gets the ed25519 signature from the signer app, if
// available.
func (s Signer) getSig() ([]byte, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tkeyclient.Dump("getSig tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return nil, fmt.Errorf("getSig NOK")
	}

	// Skip frame header, app header, and status; returning size of
	// ed25519 signature
	return rx[3 : 3+64], nil
}
