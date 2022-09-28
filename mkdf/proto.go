// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package mkdf

import (
	"encoding/hex"
	"fmt"
	"io"
)

type Endpoint byte

const (
	// destAFPGA endpoint = 1
	DestFW  Endpoint = 2
	DestApp Endpoint = 3
)

// Length of command data that follows the first 1 byte frame header
type CmdLen byte

const (
	CmdLen1   CmdLen = 0
	CmdLen4   CmdLen = 1
	CmdLen32  CmdLen = 2
	CmdLen128 CmdLen = 3
)

// Bytelen returns the number of bytes corresponding to the specific
// CmdLen value.
func (l CmdLen) Bytelen() int {
	switch l {
	case CmdLen1:
		return 1
	case CmdLen4:
		return 4
	case CmdLen32:
		return 32
	case CmdLen128:
		return 128
	}
	return 0
}

type Cmd interface {
	Code() byte
	String() string

	CmdLen() CmdLen
	Endpoint() Endpoint
}

var (
	cmdGetNameVersion = fwCmd{0x01, "cmdGetNameVersion", CmdLen1}
	rspGetNameVersion = fwCmd{0x02, "rspGetNameVersion", CmdLen32}
	cmdLoadAppSize    = fwCmd{0x03, "cmdLoadAppSize", CmdLen32}
	rspLoadAppSize    = fwCmd{0x04, "rspLoadAppSize", CmdLen4}
	cmdLoadAppData    = fwCmd{0x05, "cmdLoadAppData", CmdLen128}
	rspLoadAppData    = fwCmd{0x06, "rspLoadAppData", CmdLen4}
	cmdRunApp         = fwCmd{0x07, "cmdRunApp", CmdLen1}
	rspRunApp         = fwCmd{0x08, "rspRunApp", CmdLen4}
	cmdGetAppDigest   = fwCmd{0x09, "cmdGetAppDigest", CmdLen1}
	rspGetAppDigest   = fwCmd{0x10, "rspGetAppDigest", CmdLen128} // encoded as 0x10 by typo
	cmdLoadUSS        = fwCmd{0x0a, "cmdLoadUSS", CmdLen128}
	rspLoadUSS        = fwCmd{0x0b, "rspLoadUSS", CmdLen4}
)

type fwCmd struct {
	code   byte
	name   string
	cmdLen CmdLen
}

func (c fwCmd) Code() byte {
	return c.code
}

func (c fwCmd) CmdLen() CmdLen {
	return c.cmdLen
}

func (c fwCmd) Endpoint() Endpoint {
	return DestFW
}

func (c fwCmd) String() string {
	return c.name
}

type FramingHdr struct {
	ID       byte
	Endpoint Endpoint
	CmdLen   CmdLen
}

func parseframe(b byte) (FramingHdr, error) {
	var f FramingHdr

	if b&0x80 != 0 {
		return f, fmt.Errorf("bad version")
	}
	if b&0x4 != 0 {
		return f, fmt.Errorf("must be zero")
	}

	f.ID = byte((uint32(b) & 0x60) >> 5)
	f.Endpoint = Endpoint((b & 0x18) >> 3)
	f.CmdLen = CmdLen(b & 0x3)

	return f, nil
}

// NewFrameBuf() allocates a buffer with the appropriate size for the
// command in cmd, including the framing protocol header. The header
// byte is generated and placed in the first byte of the returned
// buffer. The cmd parameter is also used to get the endpoint and
// command length for the header. The command code from cmd is also
// placed in the second byte in the buffer.
//
// Header:
// Bit [7] (1 bit). Reserved - possible protocol version.
// Bits [6..5] (2 bits). Frame ID tag.
//
// Bits [4..3] (2 bits). Endpoint number.
//
//	HW in application_fpga
//	FW in application_fpga
//	SW (application) in application_fpga
//
// Bit [2] (1 bit). Unused. MUST be zero.
// Bits [1..0] (2 bits). Command data length.
//
//	1 byte
//	4 bytes
//	32 bytes
//	128 bytes
//
// Note that the number of bytes indicated by the command data length
// field does **not** include the command header byte. This means that
// a complete command frame, with a header indicating a data length of
// 128 bytes, is 129 bytes in length.
func NewFrameBuf(cmd Cmd, id int) ([]byte, error) {
	if id > 3 {
		return nil, fmt.Errorf("bad id")
	}
	if cmd.Endpoint() > 3 {
		return nil, fmt.Errorf("bad endpoint")
	}
	if cmd.CmdLen() > 3 {
		return nil, fmt.Errorf("bad cmdlen")
	}

	// Make a buffer with frame header + cmdLen payload
	tx := make([]byte, 1+cmd.CmdLen().Bytelen())
	tx[0] = (byte(id) << 5) | (byte(cmd.Endpoint()) << 3) | byte(cmd.CmdLen())

	// Set command code
	tx[1] = cmd.Code()

	return tx, nil
}

// Dump() hexdumps data in d with an explaining string s first. It
// assumes the data in d corresponds to the framing protocol header
// and firmware data.
func Dump(s string, d []byte) {
	hdr, err := parseframe(d[0])
	if err != nil {
		le.Printf("%s (header Unpack error: %s):\n%s", s, err, hex.Dump(d))
		return
	}
	le.Printf("%s (FrameLen: 1+%d):\n%s", s, hdr.CmdLen.Bytelen(), hex.Dump(d))
}

func (tk TillitisKey) Write(d []byte) error {
	_, err := tk.conn.Write(d)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	return nil
}

// ReadFrame() reads a response in the framing protocol. Using
// expectedResp it checks that the header's response code, length, and
// endpoint is what's expected. The expectedID is also checked against
// the ID in the header. It returns the framing protocol header,
// payload, and any error separately.
func (tk TillitisKey) ReadFrame(expectedResp Cmd, expectedID int) (FramingHdr, []byte, error) {
	var hdr FramingHdr

	if expectedID > 3 {
		return hdr, nil, fmt.Errorf("bad expected ID")
	}
	if expectedResp.Endpoint() > 3 {
		return hdr, nil, fmt.Errorf("bad expected endpoint")
	}
	if expectedResp.CmdLen() > 3 {
		return hdr, nil, fmt.Errorf("bad expected cmdlen")
	}

	// Try to read the single header byte; the Read() will any set
	// timeout. The io.ReadFull() below overrides any timeout.
	rxHdr := make([]byte, 1)
	n, err := tk.conn.Read(rxHdr)
	if err != nil {
		return hdr, nil, fmt.Errorf("Read: %w", err)
	}
	if n == 0 {
		return hdr, nil, fmt.Errorf("Read timeout")
	}

	hdr, err = parseframe(rxHdr[0])
	if err != nil {
		return hdr, nil, fmt.Errorf("Couldn't parse framing header: %w", err)
	}

	if hdr.CmdLen != expectedResp.CmdLen() {
		return hdr, nil, fmt.Errorf("Framing: Expected len %v, got %v", expectedResp.CmdLen(), hdr.CmdLen)
	}

	if hdr.Endpoint != expectedResp.Endpoint() {
		return hdr, nil, fmt.Errorf("Message not meant for us: dest %v", hdr.Endpoint)
	}
	if hdr.ID != byte(expectedID) {
		return hdr, nil, fmt.Errorf("Expected ID %d, got %d", expectedID, hdr.ID)
	}

	rxPayload := make([]byte, expectedResp.CmdLen().Bytelen())
	if _, err = io.ReadFull(tk.conn, rxPayload); err != nil {
		return hdr, nil, fmt.Errorf("ReadFull: %w", err)
	}

	if rxPayload[0] != expectedResp.Code() {
		return hdr, nil, fmt.Errorf("Expected %s, got 0x%x", expectedResp, rxPayload[0])
	}

	return hdr, rxPayload, nil
}
