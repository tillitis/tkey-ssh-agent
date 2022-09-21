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

type fwCmd byte

const (
	cmdGetNameVersion fwCmd = 0x01
	rspGetNameVersion fwCmd = 0x02
	cmdLoadAppSize    fwCmd = 0x03
	rspLoadAppSize    fwCmd = 0x04
	cmdLoadAppData    fwCmd = 0x05
	rspLoadAppData    fwCmd = 0x06
	cmdRunApp         fwCmd = 0x07
	rspRunApp         fwCmd = 0x08
	cmdGetAppDigest   fwCmd = 0x09
	rspGetAppDigest   fwCmd = 0x10
)

func (f fwCmd) String() string {
	switch f {
	case cmdGetNameVersion:
		return "cmdGetNameVersion"

	case rspGetNameVersion:
		return "rspGetNameVersion"

	case cmdLoadAppSize:
		return "cmdLoadAppSize"

	case rspLoadAppSize:
		return "rspLoadAppSize"

	case cmdLoadAppData:
		return "cmdLoadAppData"

	case rspLoadAppData:
		return "rspLoadAppData"

	case cmdRunApp:
		return "cmdRunApp"

	case rspRunApp:
		return "rspRunApp"

	case cmdGetAppDigest:
		return "cmdGetAppDigest"

	case rspGetAppDigest:
		return "rspGetAppDigest"

	default:
		return "Unknown FW code"
	}
}

type FramingHdr struct {
	ID       byte
	Endpoint Endpoint
	CmdLen   CmdLen
}

// FrameLen returns length in bytes of a complete frame, including
// header byte and cmdlen bytes.
func (f *FramingHdr) FrameLen() int {
	// XXX Could try GenframeBuf() first to ensure valid
	return 1 + f.CmdLen.Bytelen()
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

// GenFrameBuf() generates a framing protocol header and allocates a
// buffer with the appropriate size for the command.
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
// Note that the number of bytes indicated by the command data length field
// does **not** include the command header byte. This means that a complete
// command frame, with a header indicating a data length of 128 bytes, is 129
// bytes in length.
func GenFrameBuf(id byte, endpoint Endpoint, cmdlen CmdLen) ([]byte, error) {
	if id > 3 {
		return nil, fmt.Errorf("bad id")
	}
	if endpoint > 3 {
		return nil, fmt.Errorf("bad endpoint")
	}
	if cmdlen > 3 {
		return nil, fmt.Errorf("bad cmdlen")
	}

	// Make a buffer with frame header + cmdLen payload
	tx := make([]byte, 1+cmdlen.Bytelen())
	tx[0] = (id << 5) | (byte(endpoint) << 3) | byte(cmdlen)

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

// ReadFrame() reads a response in the framing protocol of expected
// length len and endpoint as in expectedDest. It returns the framing
// protocol header, payload, and any error separately.
func (tk TillitisKey) ReadFrame(len CmdLen, expectedDest Endpoint) (FramingHdr, []byte, error) {
	var hdr FramingHdr

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

	if hdr.CmdLen != len {
		return hdr, nil, fmt.Errorf("Framing: Expected len %v, got %v", len, hdr.CmdLen)
	}

	if hdr.Endpoint != expectedDest {
		return hdr, nil, fmt.Errorf("Message not meant for us: dest %v", hdr.Endpoint)
	}

	rxPayload := make([]byte, len.Bytelen())
	if _, err = io.ReadFull(tk.conn, rxPayload); err != nil {
		return hdr, nil, fmt.Errorf("ReadFull: %w", err)
	}

	return hdr, rxPayload, nil
}
