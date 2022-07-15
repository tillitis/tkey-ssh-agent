package main

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/mullvad/mta1-mkdf-signer/mkdf"
)

type MKDFSigner struct {
	conn net.Conn
}

func NewMKDFSigner() (*MKDFSigner, error) {
	mkdf.SilenceLogging()
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
	pub, err := mkdf.GetPubkey(s.conn)
	if err != nil {
		log.Printf("mkdf.GetPubKey failed: %v\n", err)
		return nil
	}
	return ed25519.PublicKey(pub)
}

func (s *MKDFSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	signature, err := mkdf.Sign(s.conn, digest)
	if err != nil {
		log.Printf("mkdf.Sign: %v", err)
		return nil, fmt.Errorf("mkdf.Sign: %w", err)
	}
	return signature, nil
}
