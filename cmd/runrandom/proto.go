// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"

	"github.com/tillitis/tkeyclient"
)

var (
	cmdGetNameVersion = appCmd{0x01, "cmdGetNameVersion", tkeyclient.CmdLen1}
	rspGetNameVersion = appCmd{0x02, "rspGetNameVersion", tkeyclient.CmdLen32}
	cmdGetRandom      = appCmd{0x03, "cmdGetRandom", tkeyclient.CmdLen4}
	rspGetRandom      = appCmd{0x04, "rspGetRandom", tkeyclient.CmdLen128}
	cmdGetPubkey      = appCmd{0x05, "cmdGetPubkey", tkeyclient.CmdLen4}
	rspGetPubkey      = appCmd{0x06, "rspGetPubkey", tkeyclient.CmdLen128}
	cmdGetSig         = appCmd{0x07, "cmdGetSig", tkeyclient.CmdLen4}
	rspCmdSig         = appCmd{0x08, "rspCmdSig", tkeyclient.CmdLen128}
	cmdGetHash        = appCmd{0x09, "cmdGetHash", tkeyclient.CmdLen4}
	rspCmdHash        = appCmd{0x0a, "rspCmdHash", tkeyclient.CmdLen128}
)

// cmdlen - (responsecode + status)
var RandomPayloadMaxBytes = rspGetRandom.CmdLen().Bytelen() - (1 + 1)

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

type RandomGen struct {
	tk *tkeyclient.TillitisKey // A connection to a TKey
}

// New allocates a struct for communicating with the random app
// running on the TKey. You're expected to pass an existing connection
// to it, so use it like this:
//
//	tk := tkeyclient.New()
//	err := tk.Connect(port)
//	randomGen := New(tk)
func New(tk *tkeyclient.TillitisKey) RandomGen {
	var randomGen RandomGen

	randomGen.tk = tk

	return randomGen
}

// Close closes the connection to the TKey
func (s RandomGen) Close() error {
	if err := s.tk.Close(); err != nil {
		return fmt.Errorf("tk.Close: %w", err)
	}
	return nil
}

// GetAppNameVersion gets the name and version of the running app in
// the same style as the stick itself.
func (s RandomGen) GetAppNameVersion() (*tkeyclient.NameVersion, error) {
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

// GetRandom fetches random data.
func (s RandomGen) GetRandom(bytes int) ([]byte, error) {
	if bytes < 1 || bytes > RandomPayloadMaxBytes {
		return nil, fmt.Errorf("number of bytes is not in [1,%d]", RandomPayloadMaxBytes)
	}

	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetRandom, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tx[2] = byte(bytes)
	tkeyclient.Dump("GetRandom tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetRandom, id)
	tkeyclient.Dump("GetRandom rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return nil, fmt.Errorf("GetRandom NOK")
	}

	ret := RandomPayloadMaxBytes
	if ret > bytes {
		ret = bytes
	}
	// Skipping frame header, app header, and status
	return rx[3 : 3+ret], nil
}

// GetPubkey fetches the public key of the signer.
func (s RandomGen) GetPubkey() ([]byte, error) {
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

func (s RandomGen) GetSignature() ([]byte, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetSig, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tkeyclient.Dump("GetSig tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspCmdSig, id)
	tkeyclient.Dump("GetSig rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	// Skipping frame header & app header
	return rx[3 : 3+64], nil
}

func (s RandomGen) GetHash() ([]byte, error) {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdGetHash, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tkeyclient.Dump("GeHash tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspCmdHash, id)
	tkeyclient.Dump("GetHash rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != 0 {
		return nil, fmt.Errorf("Return frame NOK")
	}

	// Skipping frame header & app header
	return rx[3 : 3+32], nil
}
