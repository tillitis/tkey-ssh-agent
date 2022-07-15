package main

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/mullvad/mta1signer/mta1"
)

type MKDFSigner struct {
	conn net.Conn
}

func NewMKDFSigner() (*MKDFSigner, error) {
	mta1.SilenceLogging()
	signer := &MKDFSigner{}
	err := signer.connect()
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func (s *MKDFSigner) connect() error {
	var err error
	s.conn, err = net.Dial("tcp", "localhost:4444")
	if err != nil {
		return fmt.Errorf("Dial: %w", err)
	}
	return nil
}

// implementing crypto.Signer below

func (s *MKDFSigner) Public() crypto.PublicKey {
	pub, err := mta1.GetPubkey(s.conn)
	if err != nil {
		log.Printf("mta1.GetPubKey failed: %v\n", err)
		return nil
	}
	return ed25519.PublicKey(pub)
}

func (s *MKDFSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	signature, err := mta1.Sign(s.conn, digest)
	if err != nil {
		log.Printf("mta1.Sign: %v", err)
		return nil, fmt.Errorf("mta1.Sign: %w", err)
	}
	return signature, nil
}
