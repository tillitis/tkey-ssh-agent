// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// Package mkdfsign provides a connection the the ed25519 signerapp
// running on the Tillitis Key 1. You're expected to pass an existing
// TK1 connection to it, so use it like this:
//
//	tk, err := mkdf.New(*port, *speed)
//	signer := mkdfsign.New(tk)
//
// Then use it like this to get the public key of the TK1:
//
//	pubkey, err := signer.GetPubkey()
//
// And like this to sign a message:
//
//	signature, err := signer.Sign(message)
package mkdfsign

import (
	"fmt"

	"github.com/tillitis/tillitis-key1-apps/mkdf"
)

type appCmd byte

// App protocol does not use separate response codes for each cmd (like fw
// protocol does). The cmd code is used as response code, if it was successful.
// Separate response codes for errors could be added though.
const (
	cmdGetPubkey      appCmd = 0x01
	cmdSetSize        appCmd = 0x02
	cmdSignData       appCmd = 0x03
	cmdGetSig         appCmd = 0x04
	cmdGetNameVersion appCmd = 0x05
)

type Signer struct {
	tk mkdf.TillitisKey // A connection to a Tillitis Key 1
}

// New() gets you a connection to a ed25519 signerapp running on the
// Tillitis Key 1. You're expected to pass an existing TK1 connection
// to it, so use it like this:
//
//	tk, err := mkdf.New(port, speed)
//	signer := mkdfsign.New(tk)
func New(tk mkdf.TillitisKey) Signer {
	var signer Signer

	signer.tk = tk

	return signer
}

// Close closes the connection to the TK1
func (s Signer) Close() {
	s.tk.Close()
}

// GetAppNameVersion gets the name and version of the running app in
// the same style as the stick itself.
func (s Signer) GetAppNameVersion() (*mkdf.NameVersion, error) {
	err := s.tk.SetReadTimeout(2)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	tx, err := mkdf.GenFrameBuf(2, mkdf.DestApp, mkdf.CmdLen1)
	if err != nil {
		return nil, fmt.Errorf("GenFrameBuf: %w", err)
	}

	// Set command code
	tx[1] = byte(cmdGetNameVersion)

	mkdf.Dump("GetAppNameVersion tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(mkdf.CmdLen32, mkdf.DestApp)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[0] != byte(cmdGetNameVersion) {
		return nil, fmt.Errorf("")
	}

	err = s.tk.SetReadTimeout(0)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	nameVer := &mkdf.NameVersion{}
	nameVer.Unpack(rx[1:])

	return nameVer, nil
}

// GetPubkey fetches the public key of the signer.
func (s Signer) GetPubkey() ([]byte, error) {
	tx, err := mkdf.GenFrameBuf(2, mkdf.DestApp, mkdf.CmdLen1)
	if err != nil {
		return nil, fmt.Errorf("GenFrameBuf: %w", err)
	}

	// Set command code
	tx[1] = byte(cmdGetPubkey)

	mkdf.Dump("GetPubkey tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(mkdf.CmdLen128, mkdf.DestApp)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	mkdf.Dump("GetPubKey rx", rx)
	if rx[0] != byte(cmdGetPubkey) {
		return nil, fmt.Errorf("Expected appCmdGetPubkey, got %v", rx[0])
	}

	// Skip frame header & app header, returning size of ed25519 pubkey
	return rx[1 : 1+32], nil
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
	tx, err := mkdf.GenFrameBuf(2, mkdf.DestApp, mkdf.CmdLen32)
	if err != nil {
		return fmt.Errorf("GenFrameBuf: %w", err)
	}

	// Set command code
	tx[1] = byte(cmdSetSize)

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)
	mkdf.Dump("SetAppSize tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(mkdf.CmdLen4, mkdf.DestApp)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	mkdf.Dump("SetAppSize rx", rx)
	if rx[0] != byte(cmdSetSize) {
		return fmt.Errorf("Expected appCmdSetSize, got 0x%x", rx[0])
	}

	if rx[1] != mkdf.StatusOK {
		return fmt.Errorf("SetSignSize NOK")
	}

	return nil
}

// signload loads a chunk of a message to sign and waits for a
// response from the signer.
func (s Signer) signLoad(content []byte) (int, error) {
	tx, err := mkdf.GenFrameBuf(2, mkdf.DestApp, mkdf.CmdLen128)
	if err != nil {
		return 0, fmt.Errorf("GenFrameBuf: %w", err)
	}

	// Set the command
	tx[1] = byte(cmdSignData)

	payload := make([]byte, mkdf.CmdLen128.Bytelen()-1)
	copied := copy(payload, content)

	// Add padding if not filling the payload buffer.
	if copied < len(payload) {
		padding := make([]byte, len(payload)-copied)
		copy(payload[copied:], padding)
	}

	copy(tx[2:], payload)

	mkdf.Dump("LoadSignData tx", tx)

	if err = s.tk.Write(tx); err != nil {
		return 0, fmt.Errorf("Write: %w", err)
	}

	// Wait for reply
	_, rx, err := s.tk.ReadFrame(mkdf.CmdLen4, mkdf.DestApp)
	if err != nil {
		return 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[0] != byte(cmdSignData) {
		return 0, fmt.Errorf("Expected appCmdSignData, got %v", rx[0])
	}

	if rx[1] != mkdf.StatusOK {
		return 0, fmt.Errorf("SignData NOK")
	}

	return copied, nil
}

// getSig gets the ed25519 signature from the signer app, if
// available.
func (s Signer) getSig() ([]byte, error) {
	tx, err := mkdf.GenFrameBuf(2, mkdf.DestApp, mkdf.CmdLen1)
	if err != nil {
		return nil, fmt.Errorf("GenFrameBuf: %w", err)
	}

	// Set command code
	tx[1] = byte(cmdGetSig)

	mkdf.Dump("getSig tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(mkdf.CmdLen128, mkdf.DestApp)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[0] != byte(cmdGetSig) {
		return nil, fmt.Errorf("Expected appCmdGetSig, got %v", rx[0])
	}

	// Skip app header, returning size of ed25519 signature
	return rx[1 : 1+64], nil
}
