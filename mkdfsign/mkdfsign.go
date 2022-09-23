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

var (
	cmdGetPubkey appCmd = appCmd{
		code: 0x01, cmdLen: mkdf.CmdLen1, str: "cmdGetPubkey",
	}
	rspGetPubkey appCmd = appCmd{
		code: 0x02, cmdLen: mkdf.CmdLen128, str: "rspGetPubkey",
	}
	cmdSetSize appCmd = appCmd{
		code: 0x03, cmdLen: mkdf.CmdLen32, str: "cmdSetSize",
	}
	rspSetSize appCmd = appCmd{
		code: 0x04, cmdLen: mkdf.CmdLen4, str: "rspSetSize",
	}
	cmdSignData appCmd = appCmd{
		code: 0x05, cmdLen: mkdf.CmdLen128, str: "cmdSignData",
	}
	rspSignData appCmd = appCmd{
		code: 0x06, cmdLen: mkdf.CmdLen4, str: "rspSignData",
	}
	cmdGetSig appCmd = appCmd{
		code: 0x07, cmdLen: mkdf.CmdLen1, str: "cmdGetSig",
	}
	rspGetSig appCmd = appCmd{
		code: 0x08, cmdLen: mkdf.CmdLen128, str: "rspGetSig",
	}
	cmdGetNameVersion appCmd = appCmd{
		code: 0x09, cmdLen: mkdf.CmdLen1, str: "cmdGetNameVersion",
	}
	rspGetNameVersion appCmd = appCmd{
		code: 0x0a, cmdLen: mkdf.CmdLen32, str: "rspGetNameVersion",
	}
)

type appCmd struct {
	code   byte
	str    string
	cmdLen mkdf.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() mkdf.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() mkdf.Endpoint {
	return mkdf.DestApp
}

func (c appCmd) String() string {
	return c.str
}

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
func (s Signer) Close() error {
	if err := s.tk.Close(); err != nil {
		return fmt.Errorf("tk.Close: %w", err)
	}
	return nil
}

// GetAppNameVersion gets the name and version of the running app in
// the same style as the stick itself.
func (s Signer) GetAppNameVersion() (*mkdf.NameVersion, error) {
	id := 2
	tx, err := mkdf.NewFrameBuf(cmdGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	mkdf.Dump("GetAppNameVersion tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	err = s.tk.SetReadTimeout(2)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(rspGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
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
	id := 2
	tx, err := mkdf.NewFrameBuf(cmdGetPubkey, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	mkdf.Dump("GetPubkey tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(rspGetPubkey, id)
	mkdf.Dump("GetPubKey rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
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
	id := 2
	tx, err := mkdf.NewFrameBuf(cmdSetSize, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)
	mkdf.Dump("SetAppSize tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(rspSetSize, id)
	mkdf.Dump("SetAppSize rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[1] != mkdf.StatusOK {
		return fmt.Errorf("SetSignSize NOK")
	}

	return nil
}

// signload loads a chunk of a message to sign and waits for a
// response from the signer.
func (s Signer) signLoad(content []byte) (int, error) {
	id := 2
	tx, err := mkdf.NewFrameBuf(cmdSignData, id)
	if err != nil {
		return 0, fmt.Errorf("NewFrameBuf: %w", err)
	}

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
	_, rx, err := s.tk.ReadFrame(rspSignData, id)
	if err != nil {
		return 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[1] != mkdf.StatusOK {
		return 0, fmt.Errorf("SignData NOK")
	}

	return copied, nil
}

// getSig gets the ed25519 signature from the signer app, if
// available.
func (s Signer) getSig() ([]byte, error) {
	id := 2
	tx, err := mkdf.NewFrameBuf(cmdGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	mkdf.Dump("getSig tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	_, rx, err := s.tk.ReadFrame(rspGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	// Skip app header, returning size of ed25519 signature
	return rx[1 : 1+64], nil
}
