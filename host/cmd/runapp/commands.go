package main

import (
	"fmt"
	"net"
)

type appSize struct {
	Hdr  Frame
	Size int
}

func (a *appSize) pack() ([]byte, error) {
	buf := make([]byte, a.Hdr.Len()+1)
	var err error

	// Frame header
	buf[0], err = a.Hdr.pack()
	if err != nil {
		return nil, err
	}

	// Append command code
	buf[1] = fwCmdLoadAppSize

	// Append size
	buf[2] = byte(a.Size)
	buf[3] = byte(a.Size >> 8)
	buf[4] = byte(a.Size >> 16)
	buf[5] = byte(a.Size >> 24)

	return buf, nil
}

func SetAppSize(c net.Conn, size int) error {
	appsize := appSize{
		Hdr: Frame{
			Id:       2,
			Endpoint: DestFW,
			MsgLen:   frameLen32,
		},
		Size: size,
	}

	tx, err := appsize.pack()
	if err != nil {
		return err
	}

	dump("SetAppSize tx:", tx)
	xmit(c, tx)

	rx, err := fwrecv(c, fwRspLoadAppSize, appsize.Hdr.Id, frameLen4)
	if rx[2] != 0 {
		return fmt.Errorf("SetAppSize NOK")
	}

	return nil
}

type appData struct {
	Hdr  Frame
	Data [63]byte
}

func (a *appData) Copy(content []byte) {
	copied := copy(a.Data[:], content)

	// Add padding if not filling the frame.
	if copied < 63 {
		padding := make([]byte, 63-copied)
		copy(a.Data[copied:], padding)
	}
}

func (a *appData) pack() ([]byte, error) {
	tx := make([]byte, a.Hdr.Len()+1)
	var err error

	// Frame header
	tx[0], err = a.Hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = fwCmdLoadAppData

	copy(tx[2:], a.Data[:])

	return tx, nil
}

func LoadAppData(c net.Conn, content []byte) error {
	appdata := appData{
		Hdr: Frame{
			Id:       2,
			Endpoint: DestFW,
			MsgLen:   frameLen64,
		},
	}

	appdata.Copy(content)
	tx, err := appdata.pack()

	dump("LoadAppData tx:", tx)
	xmit(c, tx)

	// Wait for reply
	rx, err := fwrecv(c, fwRspLoadAppData, appdata.Hdr.Id, frameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("LoadAppData NOK")
	}

	return nil
}

func GetAppDigest(c net.Conn) ([32]byte, error) {
	var md [32]byte

	hdr := Frame{
		Id:       2,
		Endpoint: DestFW,
		MsgLen:   frameLen1,
	}

	// Check the digest
	tx, err := PackSimple(hdr, fwCmdGetAppDigest)
	if err != nil {
		return md, fmt.Errorf("packing packet: %v", err)
	}

	dump("GetDigest tx:", tx)
	xmit(c, tx)

	rx, err := fwrecv(c, fwRspGetAppDigest, hdr.Id, frameLen64)
	if err != nil {
		return md, err
	}

	copy(md[:], rx)

	return md, nil

}

func RunApp(c net.Conn) error {
	hdr := Frame{
		Id:       2,
		Endpoint: DestFW,
		MsgLen:   frameLen1,
	}

	// Check the digest
	tx, err := PackSimple(hdr, fwCmdRunApp)
	if err != nil {
		return nil
	}

	dump("RunApp tx:", tx)
	xmit(c, tx)

	rx, err := fwrecv(c, fwRspRunApp, hdr.Id, frameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("RunApp NOK")
	}

	return nil
}

func GetPubkey(c net.Conn) ([]byte, error) {
	hdr := Frame{
		Id:       2,
		Endpoint: DestApp,
		MsgLen:   frameLen1,
	}

	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	tx[1] = byte(appCmdGetPubkey)

	dump("GetPubkey tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return nil, err
	}

	dump(" rx:", rx)

	// Skip frame header
	return rx[1:], nil
}

type signData struct {
	Hdr  Frame
	Data [63]byte
}

func (a *signData) Copy(content []byte) {
	copied := copy(a.Data[:], content)

	// Add padding if not filling the frame.
	if copied < 63 {
		padding := make([]byte, 63-copied)
		copy(a.Data[copied:], padding)
	}
}

func (a *signData) pack() ([]byte, error) {
	tx := make([]byte, a.Hdr.Len()+1)
	var err error

	// Frame header
	tx[0], err = a.Hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = appCmdSignData

	copy(tx[2:], a.Data[:])

	return tx, nil
}

type signSize struct {
	Hdr  Frame
	Size int
}

func (a *signSize) pack() ([]byte, error) {
	buf := make([]byte, a.Hdr.Len()+1)
	var err error

	// Frame header
	buf[0], err = a.Hdr.pack()
	if err != nil {
		return nil, err
	}

	// Append command code
	buf[1] = appCmdSetSize

	// Append size
	buf[2] = byte(a.Size)
	buf[3] = byte(a.Size >> 8)
	buf[4] = byte(a.Size >> 16)
	buf[5] = byte(a.Size >> 24)

	return buf, nil
}

func SignSetSize(c net.Conn, size int) error {
	signsize := signSize{
		Hdr: Frame{
			Id:       2,
			Endpoint: DestApp,
			MsgLen:   frameLen32,
		},
		Size: size,
	}

	tx, err := signsize.pack()
	if err != nil {
		return err
	}

	dump("SignSetSize tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return err
	}

	dump(" rx:", rx)
	if rx[1] != 0 {
		return fmt.Errorf("SignSetSize NOK")
	}

	return nil
}

func SignLoad(c net.Conn, data []byte) error {
	signdata := signData{
		Hdr: Frame{
			Id:       2,
			Endpoint: DestApp,
			MsgLen:   frameLen64,
		},
	}

	signdata.Copy(data)
	tx, err := signdata.pack()

	dump("SignData tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return err
	}

	dump(" rx:", rx)

	return nil
}

func GetSig(c net.Conn) ([]byte, error) {
	hdr := Frame{
		Id:       2,
		Endpoint: DestApp,
		MsgLen:   frameLen1,
	}

	var err error

	tx := make([]byte, hdr.Len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	tx[1] = byte(appCmdGetSig)

	dump("GetSig tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return nil, err
	}

	dump(" rx:", rx)

	// Skip frame header
	return rx[1:], nil
}
