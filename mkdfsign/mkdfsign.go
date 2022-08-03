package mkdfsign

import (
	"fmt"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
	"github.com/tarm/serial"
)

type appCmd byte

const (
	appCmdGetPubkey      appCmd = 0x01
	appCmdSetSize        appCmd = 0x02
	appCmdSignData       appCmd = 0x03
	appCmdGetSig         appCmd = 0x04
	appCmdGetNameVersion appCmd = 0x05
)

func GetAppNameVersion(c *serial.Port) (*mkdf.NameVersion, error) {
	hdr := mkdf.Frame{
		ID:       2,
		Endpoint: mkdf.DestApp,
		MsgLen:   mkdf.FrameLen1,
	}

	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}
	tx[1] = byte(appCmdGetNameVersion)

	mkdf.Dump("GetAppNameVersion tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return nil, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := mkdf.Recv(c)
	if err != nil {
		return nil, fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)

	nameVer := &mkdf.NameVersion{}
	// Skip frame header
	nameVer.Unpack(rx[1:])

	return nameVer, nil
}

func GetPubkey(c *serial.Port) ([]byte, error) {
	hdr := mkdf.Frame{
		ID:       2,
		Endpoint: mkdf.DestApp,
		MsgLen:   mkdf.FrameLen1,
	}

	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}
	tx[1] = byte(appCmdGetPubkey)

	mkdf.Dump("GetPubkey tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return nil, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := mkdf.Recv(c)
	if err != nil {
		return nil, fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)

	// Skip frame header
	return rx[1:], nil
}

func Sign(conn *serial.Port, data []byte) ([]byte, error) {
	err := signSetSize(conn, len(data))
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(data); i += 63 {
		err = signLoad(conn, data[i:])
		if err != nil {
			return nil, err
		}
	}

	signature, err := getSig(conn)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

type signData struct {
	hdr  mkdf.Frame
	data [63]byte
}

func (a *signData) copy(content []byte) {
	copied := copy(a.data[:], content)

	// Add padding if not filling the frame.
	if copied < 63 {
		padding := make([]byte, 63-copied)
		copy(a.data[copied:], padding)
	}
}

func (a *signData) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.Len()+1)
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}

	tx[1] = byte(appCmdSignData)

	copy(tx[2:], a.data[:])

	return tx, nil
}

type signSize struct {
	hdr  mkdf.Frame
	size int
}

func (a *signSize) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.Len()+1)
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}

	// Append command code
	tx[1] = byte(appCmdSetSize)

	// Append size
	tx[2] = byte(a.size)
	tx[3] = byte(a.size >> 8)
	tx[4] = byte(a.size >> 16)
	tx[5] = byte(a.size >> 24)

	return tx, nil
}

func signSetSize(c *serial.Port, size int) error {
	signsize := signSize{
		hdr: mkdf.Frame{
			ID:       2,
			Endpoint: mkdf.DestApp,
			MsgLen:   mkdf.FrameLen32,
		},
		size: size,
	}

	tx, err := signsize.pack()
	if err != nil {
		return err
	}

	mkdf.Dump("SignSetSize tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := mkdf.Recv(c)
	if err != nil {
		return fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)
	if rx[1] != 0 {
		return fmt.Errorf("SignSetSize NOK (%d)", rx[1])
	}

	return nil
}

func signLoad(c *serial.Port, data []byte) error {
	signdata := signData{
		hdr: mkdf.Frame{
			ID:       2,
			Endpoint: mkdf.DestApp,
			MsgLen:   mkdf.FrameLen64,
		},
	}

	signdata.copy(data)
	tx, err := signdata.pack()
	if err != nil {
		return err
	}

	mkdf.Dump("SignData tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := mkdf.Recv(c)
	if err != nil {
		return fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)

	return nil
}

func getSig(c *serial.Port) ([]byte, error) {
	hdr := mkdf.Frame{
		ID:       2,
		Endpoint: mkdf.DestApp,
		MsgLen:   mkdf.FrameLen1,
	}

	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}
	tx[1] = byte(appCmdGetSig)

	mkdf.Dump("GetSig tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return nil, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := mkdf.Recv(c)
	if err != nil {
		return nil, fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)

	// Skip frame header
	return rx[1:], nil
}
