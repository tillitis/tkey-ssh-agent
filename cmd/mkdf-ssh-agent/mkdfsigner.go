package main

import (
	"crypto"
	"crypto/ed25519"
	"errors"
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

func (s *MKDFSigner) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error) {
	// The Ed25519 signature must be made over unhashed message. See:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/crypto/ed25519/ed25519.go;l=80
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("message must not be hashed")
	}

	signature, err := mkdf.Sign(s.conn, message)
	if err != nil {
		log.Printf("mkdf.Sign: %v", err)
		return nil, fmt.Errorf("mkdf.Sign: %w", err)
	}
	return signature, nil
}
