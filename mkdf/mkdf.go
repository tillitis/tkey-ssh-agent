package mkdf

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/tarm/serial"
	"golang.org/x/crypto/blake2s"
)

var ll = log.New(os.Stdout, "", 0)

func SilenceLogging() {
	ll.SetOutput(ioutil.Discard)
}

type NameVersion struct {
	Name0   string
	Name1   string
	Version uint32
}

func (n *NameVersion) Unpack(raw []byte) {
	n.Name0 = fmt.Sprintf("%c%c%c%c", raw[3], raw[2], raw[1], raw[0])
	n.Name1 = fmt.Sprintf("%c%c%c%c", raw[7], raw[6], raw[5], raw[4])
	n.Version = binary.LittleEndian.Uint32(raw[8:12])
}

func GetNameVersion(c *serial.Port) (*NameVersion, error) {
	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		MsgLen:   FrameLen1,
	}

	tx, err := packSimple(hdr, fwCmdGetNameVersion)
	if err != nil {
		return nil, fmt.Errorf("packSimple: %w", err)
	}

	Dump("GetNameVersion tx:", tx)
	if err = Xmit(c, tx); err != nil {
		return nil, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(c, fwRspGetNameVersion, hdr.ID, FrameLen32)
	if err != nil {
		return nil, fmt.Errorf("fwRecv: %w", err)
	}

	nameVer := &NameVersion{}
	nameVer.Unpack(rx)

	return nameVer, nil
}

func LoadAppFromFile(conn *serial.Port, fileName string) error {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}
	return LoadApp(conn, content)
}

func LoadApp(conn *serial.Port, bin []byte) error {
	binLen := len(bin)
	if binLen > 65536 {
		return fmt.Errorf("File to big")
	}

	ll.Printf("app size: %v, 0x%x, 0b%b\n", binLen, binLen, binLen)

	err := setAppSize(conn, binLen)
	if err != nil {
		return err
	}

	// Load the file
	for i := 0; i < binLen; i += 63 {
		err = loadAppData(conn, bin[i:])
		if err != nil {
			return err
		}
	}

	ll.Printf("Going to getappdigest\n")
	appDigest, err := getAppDigest(conn)
	if err != nil {
		return err
	}

	digest := blake2s.Sum256(bin)

	ll.Printf("Digest from host: \n")
	printDigest(digest)
	ll.Printf("Digest from device: \n")
	printDigest(appDigest)

	if appDigest != digest {
		return fmt.Errorf("Different digests")
	}
	ll.Printf("Same digests!\n")

	// Run the app
	ll.Printf("Running the app\n")
	return runApp(conn)
}

func setAppSize(c *serial.Port, size int) error {
	appsize := appSize{
		hdr: Frame{
			ID:       2,
			Endpoint: DestFW,
			MsgLen:   FrameLen32,
		},
		size: size,
	}

	tx, err := appsize.pack()
	if err != nil {
		return err
	}

	Dump("SetAppSize tx:", tx)
	if err = Xmit(c, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(c, fwRspLoadAppSize, appsize.hdr.ID, FrameLen4)
	if err != nil {
		return fmt.Errorf("fwRecv: %w", err)
	}
	if rx[2] != 0 {
		return fmt.Errorf("SetAppSize NOK")
	}

	return nil
}

func loadAppData(c *serial.Port, content []byte) error {
	appdata := appData{
		hdr: Frame{
			ID:       2,
			Endpoint: DestFW,
			MsgLen:   FrameLen64,
		},
	}

	appdata.copy(content)
	tx, err := appdata.pack()
	if err != nil {
		return err
	}

	Dump("LoadAppData tx:", tx)
	if err = Xmit(c, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	// Wait for reply
	rx, err := fwRecv(c, fwRspLoadAppData, appdata.hdr.ID, FrameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("LoadAppData NOK")
	}

	return nil
}

func getAppDigest(c *serial.Port) ([32]byte, error) {
	var md [32]byte

	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		MsgLen:   FrameLen1,
	}

	// Check the digest
	tx, err := packSimple(hdr, fwCmdGetAppDigest)
	if err != nil {
		return md, fmt.Errorf("packSimple: %w", err)
	}

	Dump("GetDigest tx:", tx)
	if err = Xmit(c, tx); err != nil {
		return md, fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(c, fwRspGetAppDigest, hdr.ID, FrameLen64)
	if err != nil {
		return md, err
	}

	copy(md[:], rx)

	return md, nil
}

func runApp(c *serial.Port) error {
	hdr := Frame{
		ID:       2,
		Endpoint: DestFW,
		MsgLen:   FrameLen1,
	}

	tx, err := packSimple(hdr, fwCmdRunApp)
	if err != nil {
		return fmt.Errorf("packSimple: %w", err)
	}

	Dump("RunApp tx:", tx)
	if err = Xmit(c, tx); err != nil {
		return fmt.Errorf("Xmit: %w", err)
	}

	rx, err := fwRecv(c, fwRspRunApp, hdr.ID, FrameLen4)
	if err != nil {
		return err
	}

	if rx[2] != 0 {
		return fmt.Errorf("RunApp NOK")
	}

	return nil
}

func printDigest(md [32]byte) {
	for j := 0; j < 4; j++ {
		for i := 0; i < 8; i++ {
			ll.Printf("0x%02x ", md[i+8*j])
		}
		ll.Printf("\n")
	}
	ll.Printf("\n")
}
