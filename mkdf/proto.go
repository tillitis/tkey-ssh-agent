package mkdf

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/tarm/serial"
)

type appSize struct {
	hdr  Frame
	size int
}

func (a *appSize) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.Len()+1)
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
	data [63]byte
}

func (a *appData) copy(content []byte) {
	copied := copy(a.data[:], content)

	// Add padding if not filling the frame.
	if copied < 63 {
		padding := make([]byte, 63-copied)
		copy(a.data[copied:], padding)
	}
}

func (a *appData) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.Len()+1)
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(fwCmdLoadAppData)

	copy(tx[2:], a.data[:])

	return tx, nil
}

type Endpoint byte

const (
	// destIFPGA endpoint = 0
	// destAFPGA endpoint = 1
	DestFW  Endpoint = 2
	DestApp Endpoint = 3
)

type FrameLen byte

const (
	FrameLen1  FrameLen = 0
	FrameLen4  FrameLen = 1
	FrameLen32 FrameLen = 2
	FrameLen64 FrameLen = 3
)

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
	MsgLen   FrameLen
}

func (f *Frame) Len() int {
	switch f.MsgLen {
	case FrameLen1:
		return 1
	case FrameLen4:
		return 4
	case FrameLen32:
		return 32
	case FrameLen64:
		return 64
	}

	return 0
}

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
//	64 bytes
func (f *Frame) Pack() (byte, error) {
	if f.ID > 3 {
		return 0, fmt.Errorf("bad id")
	}

	hdr := (f.ID << 5) | (byte(f.Endpoint) << 3) | byte(f.MsgLen)

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
	f.MsgLen = FrameLen(b & 0x3)

	return nil
}

// Pack a simple command with no corresponding struct.
func packSimple(hdr Frame, cmd fwCmd) ([]byte, error) {
	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.Pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(cmd)

	return tx, nil
}

func Dump(s string, d []byte) {
	le.Printf("%s\n%s", s, hex.Dump(d))
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

func fwRecv(conn *serial.Port, expectedRsp fwCmd, id byte, expectedLen FrameLen) ([]byte, error) {
	// Blocks
	rx, err := Recv(conn)
	if err != nil {
		return nil, err
	}

	Dump(" rx:", rx)

	var frame Frame

	err = frame.Unpack(rx[0])
	if err != nil {
		return nil, fmt.Errorf("frame.unpack: %w", err)
	}

	if frame.MsgLen != expectedLen {
		return nil, fmt.Errorf("incorrect length %v != expected %v", frame.MsgLen, expectedLen)
	}

	if frame.ID != id {
		return nil, fmt.Errorf("incorrect id %v != expected %v", frame.ID, id)
	}

	cmd := fwCmd(rx[1])
	le.Printf("FW code: %v\n", cmd)
	if cmd != expectedRsp {
		return nil, fmt.Errorf("incorrect response code %v != expected %v", rx[1], expectedRsp)
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
		return nil, fmt.Errorf("hdr.unpack: %w", err)
	}

	rx := make([]byte, hdr.Len()+1)
	_, err = io.ReadFull(r, rx)
	if err != nil {
		return nil, fmt.Errorf("ReadFull: %w", err)
	}

	return rx, nil
}
