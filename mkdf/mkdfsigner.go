package mkdf

import (
	"fmt"

	"github.com/tarm/serial"
)

func GetAppNameVersion(c *serial.Port) (*NameVersion, error) {
	hdr := frame{
		id:       2,
		endpoint: destApp,
		msgLen:   frameLen1,
	}

	var err error

	tx := make([]byte, hdr.len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	if err != nil {
		return nil, err
	}
	tx[1] = byte(appCmdGetNameVersion)

	dump("GetAppNameVersion tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return nil, err
	}

	dump(" rx:", rx)

	nameVer := &NameVersion{}
	// Skip frame header
	nameVer.unpack(rx[1:])

	return nameVer, nil
}

func GetPubkey(c *serial.Port) ([]byte, error) {
	hdr := frame{
		id:       2,
		endpoint: destApp,
		msgLen:   frameLen1,
	}

	var err error

	tx := make([]byte, hdr.len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	if err != nil {
		return nil, err
	}
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
	hdr  frame
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
	tx := make([]byte, a.hdr.len()+1)
	var err error

	// Frame header
	tx[0], err = a.hdr.pack()
	if err != nil {
		return nil, err
	}

	tx[1] = byte(appCmdSignData)

	copy(tx[2:], a.data[:])

	return tx, nil
}

type signSize struct {
	hdr  frame
	size int
}

func (a *signSize) pack() ([]byte, error) {
	buf := make([]byte, a.hdr.len()+1)
	var err error

	// Frame header
	buf[0], err = a.hdr.pack()
	if err != nil {
		return nil, err
	}

	// Append command code
	buf[1] = byte(appCmdSetSize)

	// Append size
	buf[2] = byte(a.size)
	buf[3] = byte(a.size >> 8)
	buf[4] = byte(a.size >> 16)
	buf[5] = byte(a.size >> 24)

	return buf, nil
}

func signSetSize(c *serial.Port, size int) error {
	signsize := signSize{
		hdr: frame{
			id:       2,
			endpoint: destApp,
			msgLen:   frameLen32,
		},
		size: size,
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
		return fmt.Errorf("SignSetSize NOK (%d)", rx[1])
	}

	return nil
}

func signLoad(c *serial.Port, data []byte) error {
	signdata := signData{
		hdr: frame{
			id:       2,
			endpoint: destApp,
			msgLen:   frameLen64,
		},
	}

	signdata.copy(data)
	tx, err := signdata.pack()
	if err != nil {
		return err
	}

	dump("SignData tx:", tx)
	xmit(c, tx)

	rx, err := recv(c)
	if err != nil {
		return err
	}

	dump(" rx:", rx)

	return nil
}

func getSig(c *serial.Port) ([]byte, error) {
	hdr := frame{
		id:       2,
		endpoint: destApp,
		msgLen:   frameLen1,
	}

	var err error

	tx := make([]byte, hdr.len()+1)

	// Frame header
	tx[0], err = hdr.pack()
	if err != nil {
		return nil, err
	}
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
