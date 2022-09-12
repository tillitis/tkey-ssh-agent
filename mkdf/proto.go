package mkdf

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/tarm/serial"
)

type Endpoint byte

const (
	// destIFPGA endpoint = 0
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
	fwCmdGetNameVersion fwCmd = 0x01
	fwRspGetNameVersion fwCmd = 0x02
	fwCmdLoadAppSize    fwCmd = 0x03
	fwRspLoadAppSize    fwCmd = 0x04
	fwCmdLoadAppData    fwCmd = 0x05
	fwRspLoadAppData    fwCmd = 0x06
	fwCmdRunApp         fwCmd = 0x07
	fwRspRunApp         fwCmd = 0x08
	fwCmdGetAppDigest   fwCmd = 0x09
	fwRspGetAppDigest   fwCmd = 0x10
)

func (f fwCmd) String() string {
	switch f {
	case fwCmdGetNameVersion:
		return "fwCmdGetNameVersion"

	case fwRspGetNameVersion:
		return "fwRspGetNameVersion"

	case fwCmdLoadAppSize:
		return "fwCmdLoadAppSize"

	case fwRspLoadAppSize:
		return "fwRspLoadAppSize"

	case fwCmdLoadAppData:
		return "fwCmdLoadAppData"

	case fwRspLoadAppData:
		return "fwRspLoadAppData"

	case fwCmdRunApp:
		return "fwCmdRunApp"

	case fwRspRunApp:
		return "fwRspRunApp"

	case fwCmdGetAppDigest:
		return "fwCmdGetAppDigest"

	case fwRspGetAppDigest:
		return "fwRspGetAppDigest"

	default:
		return "Unknown FW code"
	}
}

type Frame struct {
	ID       byte
	Endpoint Endpoint
	CmdLen   CmdLen
}

// Calculate len in bytes of a complete frame, including header byte and cmdlen
// bytes.
func (f *Frame) FrameLen() int {
	// Could try f.Pack() first to ensure valid
	return 1 + f.CmdLen.Bytelen()
}

// # Pack the frame header byte
//
// Bit [7] (1 bit). Reserved - possible protocol version.
// Bits [6..5] (2 bits). Frame ID tag.
//
// Bits [4..3] (2 bits). Endpoint number.
//
//	HW in interface_fpga
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
func (f *Frame) Pack() (byte, error) {
	if f.ID > 3 {
		return 0, fmt.Errorf("bad id")
	}
	if f.Endpoint > 3 {
		return 0, fmt.Errorf("bad endpoint")
	}
	if f.CmdLen > 3 {
		return 0, fmt.Errorf("bad cmdlen")
	}

	hdr := (f.ID << 5) | (byte(f.Endpoint) << 3) | byte(f.CmdLen)

	return hdr, nil
}

func (f *Frame) Unpack(b byte) error {
	if b&0x80 != 0 {
		return fmt.Errorf("bad version")
	}
	if b&0x4 != 0 {
		return fmt.Errorf("must be zero")
	}

	f.ID = byte((uint32(b) & 0x60) >> 5)
	f.Endpoint = Endpoint((b & 0x18) >> 3)
	f.CmdLen = CmdLen(b & 0x3)

	return nil
}

// Pack a simple command with no corresponding struct.
func packSimple(hdr Frame, cmd fwCmd) ([]byte, error) {
	var err error

	tx := make([]byte, hdr.FrameLen())

	// Frame header
	tx[0], err = hdr.Pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(cmd)

	return tx, nil
}

type appSize struct {
	hdr  Frame
	size int
}

func (a *appSize) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.FrameLen())
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, err
	}

	// Append command code
	tx[1] = byte(fwCmdLoadAppSize)

	// Append size
	tx[2] = byte(a.size)
	tx[3] = byte(a.size >> 8)
	tx[4] = byte(a.size >> 16)
	tx[5] = byte(a.size >> 24)

	return tx, nil
}

type appData struct {
	hdr  Frame
	data []byte
}

func (a *appData) copy(content []byte) int {
	copied := copy(a.data, content)
	// Add padding if not filling the payload buf.
	if copied < len(a.data) {
		padding := make([]byte, len(a.data)-copied)
		copy(a.data[copied:], padding)
	}
	return copied
}

func (a *appData) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.FrameLen())
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(fwCmdLoadAppData)

	copy(tx[2:], a.data)

	return tx, nil
}

func Dump(s string, d []byte) {
	var hdr Frame
	hdr.Unpack(d[0])
	le.Printf("%s (FrameLen: 1+%d):\n%s", s, hdr.CmdLen.Bytelen(), hex.Dump(d))
}

func Xmit(c *serial.Port, d []byte) error {
	b := bufio.NewWriter(c)
	if _, err := b.Write(d); err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	if err := b.Flush(); err != nil {
		return fmt.Errorf("Flush: %w", err)
	}
	return nil
}

func fwRecv(conn *serial.Port, expectedRsp fwCmd, id byte, expectedLen CmdLen) ([]byte, error) {
	// Blocking
	rx, err := Recv(conn)
	if err != nil {
		return nil, err
	}

	Dump(" rx", rx)

	var hdr Frame

	err = hdr.Unpack(rx[0])
	if err != nil {
		return nil, fmt.Errorf("Unpack: %w", err)
	}

	rsp := fwCmd(rx[1])
	le.Printf("FW code: %v\n", rsp)
	if rsp != expectedRsp {
		return nil, fmt.Errorf("incorrect response code %v != expected %v", rsp, expectedRsp)
	}

	if hdr.CmdLen != expectedLen {
		return nil, fmt.Errorf("incorrect length %v != expected %v", hdr.CmdLen, expectedLen)
	}

	if hdr.ID != id {
		return nil, fmt.Errorf("incorrect id %v != expected %v", hdr.ID, id)
	}

	// 0 is frame header
	// 1 is fw header
	// Return the rest
	return rx[2:], nil
}

func Recv(c *serial.Port) ([]byte, error) {
	r := bufio.NewReader(c)
	b, err := r.Peek(1)
	if err != nil {
		return nil, fmt.Errorf("Peek: %w", err)
	}
	var hdr Frame

	err = hdr.Unpack(b[0])
	if err != nil {
		return nil, fmt.Errorf("Unpack: %w", err)
	}

	rx := make([]byte, hdr.FrameLen())
	_, err = io.ReadFull(r, rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFull: %w", err)
	}

	return rx, nil
}
