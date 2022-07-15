package mta1

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
)

type appSize struct {
	hdr  frame
	size int
}

func (a *appSize) pack() ([]byte, error) {
	buf := make([]byte, a.hdr.len()+1)
	var err error

	// Frame header
	buf[0], err = a.hdr.pack()
	if err != nil {
		return nil, err
	}

	// Append command code
	buf[1] = byte(fwCmdLoadAppSize)

	// Append size
	buf[2] = byte(a.size)
	buf[3] = byte(a.size >> 8)
	buf[4] = byte(a.size >> 16)
	buf[5] = byte(a.size >> 24)

	return buf, nil
}

type appData struct {
	hdr  frame
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
	tx := make([]byte, a.hdr.len()+1)
	var err error

	// Frame header
	tx[0], err = a.hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(fwCmdLoadAppData)

	copy(tx[2:], a.data[:])

	return tx, nil
}

type endpoint byte

const (
	destIFPGA endpoint = 0
	destAFPGA endpoint = 1
	destFW    endpoint = 2
	destApp   endpoint = 3
)

type frameLen byte

const (
	frameLen1  frameLen = 0
	frameLen4  frameLen = 1
	frameLen32 frameLen = 2
	frameLen64 frameLen = 3
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

type appCmd byte

const (
	appCmdGetPubkey appCmd = 0x01
	appCmdSetSize   appCmd = 0x03
	appCmdSignData  appCmd = 0x04
	appCmdGetSig    appCmd = 0x05
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

type frame struct {
	id       byte
	endpoint endpoint
	msgLen   frameLen
}

func (f *frame) len() int {
	switch f.msgLen {
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
func (f *frame) pack() (byte, error) {
	if f.id > 3 {
		return 0, fmt.Errorf("bad id")
	}

	hdr := (f.id << 5) | (byte(f.endpoint) << 3) | byte(f.msgLen)

	return hdr, nil
}

func (h *frame) unpack(b byte) error {
	if b&0x80 != 0 {
		return fmt.Errorf("bad version")

	}
	if b&0x4 != 0 {
		return fmt.Errorf("must be zero")

	}

	h.id = byte((uint32(b) & 0x60) >> 5)
	h.endpoint = endpoint(byte(b&0x18) >> 3)
	h.msgLen = frameLen(byte(b) & 0x3)

	return nil
}

// Pack a simple command with no corresponding struct
func packSimple(hdr frame, cmd fwCmd) ([]byte, error) {
	var err error

	tx := make([]byte, hdr.len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(cmd)

	return tx, nil

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

func fwRecv(conn net.Conn, expectedRsp fwCmd, id byte, expectedLen frameLen) ([]byte, error) {
	// Blocks
	rx, err := recv(conn)
	if err != nil {
		return nil, err
	}

	dump(" rx:", rx)

	var frame frame

	err = frame.unpack(rx[0])
	if err != nil {
		return nil, err
	}

	if frame.msgLen != expectedLen {
		return nil, fmt.Errorf("incorrect length %v != expected %v", frame.msgLen, expectedLen)
	}

	if frame.id != id {
		fmt.Errorf("incorrect id %v != expected %v", frame.id, id)
	}

	cmd := fwCmd(rx[1])
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
	var hdr frame

	err = hdr.unpack(b[0])
	if err != nil {
		return nil, err
	}

	rx := make([]byte, hdr.len()+1)
	_, err = io.ReadFull(r, rx)
	if err != nil {
		return nil, err
	}

	return rx, nil
}
