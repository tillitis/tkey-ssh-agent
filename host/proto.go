package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
)

type endpoint byte

const (
	DestIfpga endpoint = 0
	DestAfgpa          = 1
	DestFW             = 2
	DestApp            = 3
)

type frameLen byte

const (
	frameLen1  frameLen = 0
	frameLen4           = 1
	frameLen32          = 2
	frameLen64          = 3
)

type fwcmd byte

const (
	fwCmdGetNameVersion fwcmd = 0x01
	fwRspGetNameVersion       = 0x02
	fwCmdLoadAppSize          = 0x03
	fwRspLoadAppSize          = 0x04
	fwCmdLoadAppData          = 0x05
	fwRspLoadAppData          = 0x06
	fwCmdRunApp               = 0x07
	fwRspRunApp               = 0x08
	fwCmdGetAppDigest         = 0x09
	fwRspGetAppDigest         = 0x10
)

type appcmd byte

const (
	appCmdGetPubkey appcmd = 0x01
	appCmdSetSize          = 0x03
	appCmdSignData         = 0x04
	appCmdGetSig           = 0x05
)

func (f fwcmd) String() string {
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
	Id       byte
	Endpoint endpoint
	MsgLen   frameLen
}

func (f *Frame) Len() int {
	switch f.MsgLen {
	case frameLen1:
		return 1
	case frameLen4:
		return 4
	case frameLen32:
		return 32
	case frameLen64:
		return 64
	}

	return 0
}

// Bit [7] (1 bit). Reserved - possible protocol version.
// Bits [6..5] (2 bits). Frame ID tag.
// Bits [4..3] (2 bits). Endpoint number.
//   HW in interface_fpga
//   HW in application_fpga
//   FW in application_fpga
//   SW (application) in application_fpga
// Bit [2] (1 bit). Unused. MUST be zero.
// Bits [1..0] (2 bits). Command data length.
//   1 byte
//   4 bytes
//   32 bytes
//   64 bytes
func (f *Frame) pack() (byte, error) {
	if f.Id > 3 {
		return 0, fmt.Errorf("bad id")
	}

	hdr := (f.Id << 5) | (byte(f.Endpoint) << 3) | byte(f.MsgLen)

	return hdr, nil
}

func (h *Frame) unpack(b byte) error {
	if b&0x80 != 0 {
		return fmt.Errorf("bad version")

	}
	if b&0x4 != 0 {
		return fmt.Errorf("must be zero")

	}

	h.Id = byte((uint32(b) & 0x60) >> 5)
	h.Endpoint = endpoint(byte(b&0x10) >> 3)

	h.MsgLen = frameLen(byte(b) & 0x3)

	return nil
}

// Pack a simple command with no corresponding struct
func PackSimple(hdr Frame, cmd fwcmd) ([]byte, error) {
	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(cmd)

	return tx, nil

}

func connect() (net.Conn, error) {
	conn, err := net.Dial("tcp", "localhost:4444")
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func dump(s string, d []byte) {
	fmt.Printf("%s\n%s", s, hex.Dump(d))
}

func xmit(c net.Conn, d []byte) {
	b := bufio.NewWriter(c)
	_, err := b.Write(d)
	if err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		panic("xmit")
	}
	err = b.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		panic("xmit")
	}
}

func fwrecv(conn net.Conn, expectedRsp fwcmd, id byte, expectedLen frameLen) ([]byte, error) {
	// Blocks
	rx, err := recv(conn)
	if err != nil {
		return nil, err
	}

	dump(" rx:", rx)

	var frame Frame

	err = frame.unpack(rx[0])
	if err != nil {
		return nil, err
	}

	if frame.MsgLen != expectedLen {
		return nil, fmt.Errorf("incorrect length %v != expected %v", frame.MsgLen, expectedLen)
	}

	if frame.Id != id {
		fmt.Errorf("incorrect id %v != expected %v", frame.Id, id)
	}

	cmd := fwcmd(rx[1])
	fmt.Printf("FW code: %v\n", cmd)
	if cmd != expectedRsp {
		fmt.Errorf("incorrect response code %v != expected %v", rx[1], expectedRsp)
	}

	// 0 is frame header
	// 1 is fw header
	// Return the rest
	return rx[2:], nil
}

func recv(c net.Conn) ([]byte, error) {
	r := bufio.NewReader(c)
	b, err := r.Peek(1)
	if err != nil {
		return nil, err
	}
	var hdr Frame

	err = hdr.unpack(b[0])
	if err != nil {
		return nil, err
	}

	rx := make([]byte, hdr.Len()+1)
	_, err = io.ReadFull(r, rx)
	if err != nil {
		return nil, err
	}

	return rx, nil
}
