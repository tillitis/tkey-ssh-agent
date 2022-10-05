// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"

	"github.com/tillitis/tillitis-key1-apps/mkdf"
)

var (
	cmdGetNameVersion = appCmd{0x01, "cmdGetNameVersion", mkdf.CmdLen1}
	rspGetNameVersion = appCmd{0x02, "rspGetNameVersion", mkdf.CmdLen32}
	cmdGetRandom      = appCmd{0x03, "cmdGetRandom", mkdf.CmdLen4}
	rspGetRandom      = appCmd{0x04, "rspGetRandom", mkdf.CmdLen128}
)

// RSP_GET_RANDOM cmdlen - (responsecode + status)
const RandomPayloadMaxBytes = 128 - (1 + 1)

type appCmd struct {
	code   byte
	name   string
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
	return c.name
}

type randomGen struct {
	tk mkdf.TillitisKey // A connection to a Tillitis Key 1
}

// New() gets you a connection to the random app running on the
// Tillitis Key 1. You're expected to pass an existing TK1 connection
// to it, so use it like this:
//
//	tk, err := mkdf.New(port, speed)
//	randomGen := mkdfrand.New(tk)
func New(tk mkdf.TillitisKey) randomGen {
	var randomGen randomGen

	randomGen.tk = tk

	return randomGen
}

// Close closes the connection to the TK1
func (s randomGen) Close() error {
	if err := s.tk.Close(); err != nil {
		return fmt.Errorf("tk.Close: %w", err)
	}
	return nil
}

// GetAppNameVersion gets the name and version of the running app in
// the same style as the stick itself.
func (s randomGen) GetAppNameVersion() (*mkdf.NameVersion, error) {
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

	rx, _, err := s.tk.ReadFrame(rspGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	err = s.tk.SetReadTimeout(0)
	if err != nil {
		return nil, fmt.Errorf("SetReadTimeout: %w", err)
	}

	nameVer := &mkdf.NameVersion{}
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

// GetRandom fetches random data.
func (s randomGen) GetRandom(bytes int) ([]byte, error) {
	if bytes < 1 || bytes > RandomPayloadMaxBytes {
		return nil, fmt.Errorf("number of bytes is not in [1,%d]", RandomPayloadMaxBytes)
	}

	id := 2
	tx, err := mkdf.NewFrameBuf(cmdGetRandom, id)
	if err != nil {
		return nil, fmt.Errorf("NewFrameBuf: %w", err)
	}

	tx[2] = byte(bytes)
	mkdf.Dump("GetRandom tx", tx)
	if err = s.tk.Write(tx); err != nil {
		return nil, fmt.Errorf("Write: %w", err)
	}

	rx, _, err := s.tk.ReadFrame(rspGetRandom, id)
	mkdf.Dump("GetRandom rx", rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != mkdf.StatusOK {
		return nil, fmt.Errorf("GetRandom NOK")
	}

	ret := RandomPayloadMaxBytes
	if ret > bytes {
		ret = bytes
	}
	// Skipping frame header, app header, and status
	return rx[3 : 3+ret], nil
}
