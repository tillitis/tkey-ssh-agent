package mkdfsign

import (
	"fmt"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
	"github.com/tarm/serial"
)

type appCmd byte

// App protocol does not use separate response codes for each cmd (like fw
// protocol does). The cmd code is used as response code, if it was successful.
// Separate response codes for errors could be added though.
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
		CmdLen:   mkdf.CmdLen1,
	}

	var err error

	tx := make([]byte, hdr.FrameLen())

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

	rx, err := appRecv(c, appCmd(tx[1]), hdr.ID, mkdf.CmdLen32)
	if err != nil {
		return nil, fmt.Errorf("appRecv: %w", err)
	}

	nameVer := &mkdf.NameVersion{}
	// Skip frame header & app header
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

func GetPubkey(c *serial.Port) ([]byte, error) {
	hdr := mkdf.Frame{
		ID:       2,
		Endpoint: mkdf.DestApp,
		CmdLen:   mkdf.CmdLen1,
	}

	var err error

	tx := make([]byte, hdr.FrameLen())

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

	rx, err := appRecv(c, appCmd(tx[1]), hdr.ID, mkdf.CmdLen128)
	if err != nil {
		return nil, fmt.Errorf("appRecv: %w", err)
	}

	// Skip frame header & app header, returning size of ed25519 pubkey
	return rx[2 : 2+32], nil
}

func Sign(conn *serial.Port, data []byte) ([]byte, error) {
	err := signSetSize(conn, len(data))
	if err != nil {
		return nil, fmt.Errorf("signSetSize: %w", err)
	}

	var offset int
	for nsent := 0; offset < len(data); offset += nsent {
		nsent, err = signLoad(conn, data[offset:])
		if err != nil {
			return nil, fmt.Errorf("signLoad: %w", err)
		}
	}
	if offset > len(data) {
		return nil, fmt.Errorf("transmitted more than expected")
	}

	signature, err := getSig(conn)
	if err != nil {
		return nil, fmt.Errorf("getSig: %w", err)
	}

	return signature, nil
}

type signSize struct {
	hdr  mkdf.Frame
	size int
}

func (a *signSize) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.FrameLen())
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
			CmdLen:   mkdf.CmdLen32,
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

	rx, err := appRecv(c, appCmd(tx[1]), signsize.hdr.ID, mkdf.CmdLen4)
	if err != nil {
		return fmt.Errorf("appRecv: %w", err)
	}

	if rx[2] != mkdf.StatusOK {
		return fmt.Errorf("signSetSize NOK (%d)", rx[2])
	}

	return nil
}

type signData struct {
	hdr  mkdf.Frame
	data []byte
}

func (a *signData) copy(content []byte) int {
	copied := copy(a.data, content)
	// Add padding if not filling the payload buf.
	if copied < len(a.data) {
		padding := make([]byte, len(a.data)-copied)
		copy(a.data[copied:], padding)
	}
	return copied
}

func (a *signData) pack() ([]byte, error) {
	tx := make([]byte, a.hdr.FrameLen())
	var err error

	// Frame header
	tx[0], err = a.hdr.Pack()
	if err != nil {
		return nil, fmt.Errorf("Pack: %w", err)
	}

	tx[1] = byte(appCmdSignData)

	copy(tx[2:], a.data)

	return tx, nil
}

func signLoad(c *serial.Port, data []byte) (int, error) {
	cmdLen := mkdf.CmdLen128
	signdata := signData{
		hdr: mkdf.Frame{
			ID:       2,
			Endpoint: mkdf.DestApp,
			CmdLen:   cmdLen,
		},
		// Payload len is cmdlen minus the app cmd byte
		data: make([]byte, cmdLen.Bytelen()-1),
	}

	nsent := signdata.copy(data)

	tx, err := signdata.pack()
	if err != nil {
		return 0, err
	}

	mkdf.Dump("SignData tx:", tx)
	if err = mkdf.Xmit(c, tx); err != nil {
		return 0, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := appRecv(c, appCmd(tx[1]), signdata.hdr.ID, mkdf.CmdLen4)
	if err != nil {
		return 0, fmt.Errorf("appRecv: %w", err)
	}

	if rx[2] != mkdf.StatusOK {
		return 0, fmt.Errorf("signData NOK (%d)", rx[2])
	}

	return nsent, nil
}

func getSig(c *serial.Port) ([]byte, error) {
	hdr := mkdf.Frame{
		ID:       2,
		Endpoint: mkdf.DestApp,
		CmdLen:   mkdf.CmdLen1,
	}

	var err error

	tx := make([]byte, hdr.FrameLen())

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

	rx, err := appRecv(c, appCmd(tx[1]), hdr.ID, mkdf.CmdLen128)
	if err != nil {
		return nil, fmt.Errorf("appRecv: %w", err)
	}

	// Skip frame header & app header, returning size of ed25519 signature
	return rx[2 : 2+64], nil
}

func appRecv(conn *serial.Port, expectedRsp appCmd, id byte, expectedLen mkdf.CmdLen) ([]byte, error) {
	rx, err := mkdf.Recv(conn)
	if err != nil {
		return nil, fmt.Errorf("Recv: %w", err)
	}

	mkdf.Dump(" rx:", rx)

	var hdr mkdf.Frame

	err = hdr.Unpack(rx[0])
	if err != nil {
		return nil, fmt.Errorf("Unpack: %w", err)
	}

	rsp := appCmd(rx[1])
	if rsp != expectedRsp {
		return nil, fmt.Errorf("incorrect response code %v != expected %v", rsp, expectedRsp)
	}

	if hdr.CmdLen != expectedLen {
		return nil, fmt.Errorf("incorrect length %v != expected %v", hdr.CmdLen, expectedLen)
	}

	if hdr.ID != id {
		return nil, fmt.Errorf("incorrect id %v != expected %v", hdr.ID, id)
	}

	return rx, nil
}
