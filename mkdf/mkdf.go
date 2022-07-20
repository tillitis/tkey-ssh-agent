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

var ll *log.Logger

func init() {
	ll = log.New(os.Stdout, "", 0)
}

func SilenceLogging() {
	ll.SetOutput(ioutil.Discard)
}

type NameVersion struct {
	Name0   string
	Name1   string
	Version uint32
}

func (n *NameVersion) unpack(raw []byte) {
	n.Name0 = fmt.Sprintf("%c%c%c%c", raw[3], raw[2], raw[1], raw[0])
	n.Name1 = fmt.Sprintf("%c%c%c%c", raw[7], raw[6], raw[5], raw[4])
	n.Version = binary.LittleEndian.Uint32(raw[8:12])
}

func GetNameVersion(c *serial.Port) (*NameVersion, error) {
	hdr := frame{
		id:       2,
		endpoint: destFW,
		msgLen:   frameLen1,
	}

	tx, err := packSimple(hdr, fwCmdGetNameVersion)
	if err != nil {
		return nil, fmt.Errorf("packSimple: %w", err)
	}

	dump("GetNameVersion tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspGetNameVersion, hdr.id, frameLen32)
	if err != nil {
		return nil, fmt.Errorf("fwRecv: %w", err)
	}

	nameVer := &NameVersion{}
	nameVer.unpack(rx)

	return nameVer, nil
}

func LoadApp(conn *serial.Port, fileName string) error {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}

	contentlen := len(content)
	if contentlen > 65536 {
		return fmt.Errorf("File to big")
	}

	ll.Printf("app size: %v, 0x%x, 0b%b\n", contentlen, contentlen, contentlen)

	err = setAppSize(conn, contentlen)
	if err != nil {
		return err
	}

	// Load the file
	for i := 0; i < contentlen; i += 63 {
		err = loadAppData(conn, content[i:])
		if err != nil {
			return err
		}
	}

	ll.Printf("Going to getappdigest\n")
	appDigest, err := getAppDigest(conn)
	if err != nil {
		return err
	}

	digest := blake2s.Sum256(content)

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
		hdr: frame{
			id:       2,
			endpoint: destFW,
			msgLen:   frameLen32,
		},
		size: size,
	}

	tx, err := appsize.pack()
	if err != nil {
		return err
	}

	dump("SetAppSize tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspLoadAppSize, appsize.hdr.id, frameLen4)
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
		hdr: frame{
			id:       2,
			endpoint: destFW,
			msgLen:   frameLen64,
		},
	}

	appdata.copy(content)
	tx, err := appdata.pack()
	if err != nil {
		return err
	}

	dump("LoadAppData tx:", tx)
	xmit(c, tx)

	// Wait for reply
	rx, err := fwRecv(c, fwRspLoadAppData, appdata.hdr.id, frameLen4)
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

	hdr := frame{
		id:       2,
		endpoint: destFW,
		msgLen:   frameLen1,
	}

	// Check the digest
	tx, err := packSimple(hdr, fwCmdGetAppDigest)
	if err != nil {
		return md, fmt.Errorf("packSimple: %w", err)
	}

	dump("GetDigest tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspGetAppDigest, hdr.id, frameLen64)
	if err != nil {
		return md, err
	}

	copy(md[:], rx)

	return md, nil
}

func runApp(c *serial.Port) error {
	hdr := frame{
		id:       2,
		endpoint: destFW,
		msgLen:   frameLen1,
	}

	tx, err := packSimple(hdr, fwCmdRunApp)
	if err != nil {
		return fmt.Errorf("packSimple: %w", err)
	}

	dump("RunApp tx:", tx)
	xmit(c, tx)

	rx, err := fwRecv(c, fwRspRunApp, hdr.id, frameLen4)
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
