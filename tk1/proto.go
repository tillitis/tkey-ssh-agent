// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package tk1

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
	cmdGetNameVersion   = fwCmd{0x01, "cmdGetNameVersion", CmdLen1}
	rspGetNameVersion   = fwCmd{0x02, "rspGetNameVersion", CmdLen32}
	cmdLoadApp          = fwCmd{0x03, "cmdLoadApp", CmdLen128}
	rspLoadApp          = fwCmd{0x04, "rspLoadApp", CmdLen4}
	cmdLoadAppData      = fwCmd{0x05, "cmdLoadAppData", CmdLen128}
	rspLoadAppData      = fwCmd{0x06, "rspLoadAppData", CmdLen4}
	rspLoadAppDataReady = fwCmd{0x07, "rspLoadAppDataReady", CmdLen128}
	cmdGetUDI           = fwCmd{0x08, "cmdGetUDI", CmdLen1}
	rspGetUDI           = fwCmd{0x09, "rspGetUDI", CmdLen32}
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
	ID            byte
	Endpoint      Endpoint
	CmdLen        CmdLen
	ResponseNotOK bool
}

func parseframe(b byte) (FramingHdr, error) {
	var f FramingHdr

	if (b & 0b1000_0000) != 0 {
		return f, fmt.Errorf("reserved bit #7 is not zero")
	}

	// If bit #2 is set
	if (b & 0b0000_0100) != 0 {
		f.ResponseNotOK = true
	}

	f.ID = byte((b & 0b0110_0000) >> 5)
	f.Endpoint = Endpoint((b & 0b0001_1000) >> 3)
	f.CmdLen = CmdLen(b & 0b0000_0011)

	return f, nil
}

// NewFrameBuf allocates a buffer with the appropriate size for the
// command in cmd, including the framing protocol header byte. The cmd
// parameter is used to get the endpoint and command length, which
// together with id parameter are encoded as the header byte. The
// header byte is placed in the first byte in the returned buffer. The
// command code from cmd is placed in the buffer's second byte.
//
// Header byte (used for both command and response frame):
//
// Bit [7] (1 bit). Reserved - possible protocol version.
//
// Bits [6..5] (2 bits). Frame ID tag.
//
// Bits [4..3] (2 bits). Endpoint number:
//
//	00 == reserved
//	01 == HW in application_fpga
//	10 == FW in application_fpga
//	11 == SW (application) in application_fpga
//
// Bit [2] (1 bit). Usage:
//
//	Command: Unused. MUST be zero.
//	Response: 0 == OK, 1 == Not OK (NOK)
//
// Bits [1..0] (2 bits). Command/Response data length:
//
//	00 == 1 byte
//	01 == 4 bytes
//	10 == 32 bytes
//	11 == 128 bytes
//
// Note that the number of bytes indicated by the command data length
// field does **not** include the header byte. This means that a
// complete command frame, with a header indicating a command length
// of 128 bytes, is 127 bytes in length.
func NewFrameBuf(cmd Cmd, id int) ([]byte, error) {
	if id > 3 {
		return nil, fmt.Errorf("frame ID must be 0..3")
	}
	if cmd.Endpoint() > 3 {
		return nil, fmt.Errorf("endpoint must be 0..3")
	}
	if cmd.CmdLen() > 3 {
		return nil, fmt.Errorf("cmdlen must be 0..3")
	}

	// Make a buffer with frame header + cmdLen payload
	tx := make([]byte, 1+cmd.CmdLen().Bytelen())
	tx[0] = (byte(id) << 5) | (byte(cmd.Endpoint()) << 3) | byte(cmd.CmdLen())

	// Set command code
	tx[1] = cmd.Code()

	return tx, nil
}

// Dump() hexdumps data in d with an explaining string s first. It
// expects d to contain the whole frame as sent on the wire, with the
// framing protocol header in the first byte.
func Dump(s string, d []byte) {
	if d == nil || len(d) == 0 {
		le.Printf("%s: no data\n", s)
		return
	}
	hdr, err := parseframe(d[0])
	if err != nil {
		le.Printf("%s (parseframe error: %s):\n", s, err)
	} else {
		le.Printf("%s (frame len: 1+%d bytes):\n", s, hdr.CmdLen.Bytelen())
	}
	le.Printf("%s", hex.Dump(d))
}

func (tk TillitisKey) Write(d []byte) error {
	_, err := tk.conn.Write(d)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	return nil
}

type constError string

func (err constError) Error() string {
	return string(err)
}

const ErrResponseStatusNotOK = constError("response status not OK")

// ReadFrame reads a response in the framing protocol. The header byte
// is first parsed. If the header has response status Not OK,
// ErrResponseStatusNotOK is returned. Header command length and
// endpoint are then checked against the expectedResp parameter,
// header ID is checked against expectedID. The response code (first
// byte after header) is also checked against the code in
// expectedResp. It returns the whole frame read, the parsed header
// byte, and any error separately.
func (tk TillitisKey) ReadFrame(expectedResp Cmd, expectedID int) ([]byte, FramingHdr, error) {
	if expectedID > 3 {
		return nil, FramingHdr{}, fmt.Errorf("frame ID to expect must be 0..3")
	}
	if expectedResp.Endpoint() > 3 {
		return nil, FramingHdr{}, fmt.Errorf("endpoint to expect must be 0..3")
	}
	if expectedResp.CmdLen() > 3 {
		return nil, FramingHdr{}, fmt.Errorf("cmdlen to expect must be 0..3")
	}

	// Try to read the single header byte
	rxHdr := make([]byte, 1)
	// Read() obeys timeout set using SetReadTimeout()
	n, err := tk.conn.Read(rxHdr)
	if err != nil {
		return nil, FramingHdr{}, fmt.Errorf("Read: %w", err)
	}
	if n == 0 {
		return nil, FramingHdr{}, fmt.Errorf("Read timeout")
	}

	hdr, err := parseframe(rxHdr[0])
	if err != nil {
		return nil, hdr, fmt.Errorf("Couldn't parse framing header: %w", err)
	}

	if hdr.ResponseNotOK {
		err = ErrResponseStatusNotOK
		// We still need to read out the data/payload, note that
		// ReadFull() overrides any timeout set using SetReadTimeout()
		rest := make([]byte, hdr.CmdLen.Bytelen())
		if _, readErr := io.ReadFull(tk.conn, rest); readErr != nil {
			// NOTE: go 1.20 has errors.Join()
			err = fmt.Errorf("%w; ReadFull: %w", ErrResponseStatusNotOK, readErr)
		}
		return nil, hdr, err
	}

	if hdr.CmdLen != expectedResp.CmdLen() {
		return nil, hdr, fmt.Errorf("Expected cmdlen %v (%d bytes), got %v (%d bytes)",
			expectedResp.CmdLen(), expectedResp.CmdLen().Bytelen(),
			hdr.CmdLen, hdr.CmdLen.Bytelen())
	}

	if hdr.Endpoint != expectedResp.Endpoint() {
		return nil, hdr, fmt.Errorf("Message not meant for us: dest %v", hdr.Endpoint)
	}
	if hdr.ID != byte(expectedID) {
		return nil, hdr, fmt.Errorf("Expected ID %d, got %d", expectedID, hdr.ID)
	}

	// Prepare a buffer with the header byte first, for returning
	rx := make([]byte, 1+expectedResp.CmdLen().Bytelen())
	rx[0] = rxHdr[0]
	// Try to read the whole rest of the frame; ReadFull() overrides
	// any timeout set using SetReadTimeout()
	if _, err = io.ReadFull(tk.conn, rx[1:]); err != nil {
		return nil, hdr, fmt.Errorf("ReadFull: %w", err)
	}

	if rx[1] != expectedResp.Code() {
		return rx, hdr, fmt.Errorf("Expected cmd code 0x%x (%s), got 0x%x", expectedResp.Code(), expectedResp, rx[1])
	}

	return rx, hdr, nil
}
