package main

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
)

type FakeHardwareSigner struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (h *FakeHardwareSigner) init() error {
	var err error
	h.pub, h.priv, err = ed25519.GenerateKey(nil)
	return fmt.Errorf("GenerateKey: %w", err)
}

func NewFakeHardwareSigner() (*FakeHardwareSigner, error) {
	signer := &FakeHardwareSigner{}
	err := signer.init()
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// implementing crypto.Signer below

func (h *FakeHardwareSigner) Public() crypto.PublicKey {
	return h.pub
}

func (h *FakeHardwareSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	signature, err := h.priv.Sign(rand, digest, opts)
	if err != nil {
		return nil, fmt.Errorf("PrivateKey.Sign: %w", err)
	}
	return signature, nil
}
