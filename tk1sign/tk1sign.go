// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// Package tk1sign provides a connection to the ed25519 signer app
// running on the TKey. You're expected to pass an existing connection
// to it, so use it like this:
//
//	tk := tk1.New()
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

	"github.com/tillitis/tillitis-key1-apps/tk1"
)

var (
	cmdGetPubkey      = appCmd{0x01, "cmdGetPubkey", tk1.CmdLen1}
	rspGetPubkey      = appCmd{0x02, "rspGetPubkey", tk1.CmdLen128}
	cmdSetSize        = appCmd{0x03, "cmdSetSize", tk1.CmdLen32}
	rspSetSize        = appCmd{0x04, "rspSetSize", tk1.CmdLen4}
	cmdSignData       = appCmd{0x05, "cmdSignData", tk1.CmdLen128}
	rspSignData       = appCmd{0x06, "rspSignData", tk1.CmdLen4}
	cmdGetSig         = appCmd{0x07, "cmdGetSig", tk1.CmdLen1}
	rspGetSig         = appCmd{0x08, "rspGetSig", tk1.CmdLen128}
	cmdGetNameVersion = appCmd{0x09, "cmdGetNameVersion", tk1.CmdLen1}
	rspGetNameVersion = appCmd{0x0a, "rspGetNameVersion", tk1.CmdLen32}
	cmdGetUDI         = appCmd{0x0b, "cmdGetUDI", tk1.CmdLen1}
	rspGetUDI         = appCmd{0x0c, "rspGetUDI", tk1.CmdLen32}
)

type appCmd struct {
	code   byte
	name   string
	cmdLen tk1.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() tk1.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() tk1.Endpoint {
	return tk1.DestApp
}

func (c appCmd) String() string {
	return c.name
}

type Signer struct {
	tk *tk1.TillitisKey // A connection to a TKey
}

// New allocates a struct for communicating with the ed25519 signer
// app running on the TKey. You're expected to pass an existing
// connection to it, so use it like this:
//
//	tk := tk1.New()
//	err := tk.Connect(port)
//	signer := tk1sign.New(tk)
func New(tk *tk1.TillitisKey) Signer {
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
func (s Signer) GetAppNameVersion() (*tk1.NameVersion, error) {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tk1.Dump("GetAppNameVersion tx", tx)
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

	nameVer := &tk1.NameVersion{}
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

// GetUDI gets the two 32-bit words of Unique Device ID (UDI),
// returning them as 16 hex characters.
func (s Signer) GetUDI() (*tk1.UDI, error) {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdGetUDI, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tk1.Dump("GetUDI tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetUDI, id)
	tk1.Dump("GetUDI rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return nil, fmt.Errorf("GetUDI NOK")
	}

	udi := &tk1.UDI{}
	udi.Unpack(rx[3 : 3+8])

	return udi, nil
}

// GetPubkey fetches the public key of the signer.
func (s Signer) GetPubkey() ([]byte, error) {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdGetPubkey, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tk1.Dump("GetPubkey tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetPubkey, id)
	tk1.Dump("GetPubKey rx", rx)
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
	tx, err := tk1.NewFrameBuf(cmdSetSize, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)
	tk1.Dump("SetAppSize tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspSetSize, id)
	tk1.Dump("SetAppSize rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return fmt.Errorf("SetSignSize NOK")
	}

	return nil
}

// signload loads a chunk of a message to sign and waits for a
// response from the signer.
func (s Signer) signLoad(content []byte) (int, error) {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdSignData, id)
	if err != nil {
		return 0, fmt.Errorf("NewFrameBuf: %w", err)
	}

	payload := make([]byte, tk1.CmdLen128.Bytelen()-1)
	copied := copy(payload, content)

	// Add padding if not filling the payload buffer.
	if copied < len(payload) {
		padding := make([]byte, len(payload)-copied)
		copy(payload[copied:], padding)
	}

	copy(tx[2:], payload)

	tk1.Dump("LoadSignData tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return 0, fmt.Errorf("Write: %w", err)
	}

	// Wait for reply
	rx, _, err := s.tk.ReadFrame(rspSignData, id)
	if err != nil {
		return 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return 0, fmt.Errorf("SignData NOK")
	}

	return copied, nil
}

// getSig gets the ed25519 signature from the signer app, if
// available.
func (s Signer) getSig() ([]byte, error) {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tk1.Dump("getSig tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return nil, fmt.Errorf("getSig NOK")
	}

	// Skip frame header, app header, and status; returning size of
	// ed25519 signature
	return rx[3 : 3+64], nil
}
