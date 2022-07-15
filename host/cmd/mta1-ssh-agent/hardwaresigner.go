package main

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
)

type HardwareSigner struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (h *HardwareSigner) init() error {
	var err error
	h.pub, h.priv, err = ed25519.GenerateKey(nil)
	return fmt.Errorf("GenerateKey: %w", err)
}

func NewHardwareSigner() (*HardwareSigner, error) {
	hwSigner := &HardwareSigner{}
	err := hwSigner.init()
	if err != nil {
		return nil, err
	}
	return hwSigner, nil
}

// implementing crypto.Signer below

func (h *HardwareSigner) Public() crypto.PublicKey {
	return h.pub
}

func (h *HardwareSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	signature, err := h.priv.Sign(rand, digest, opts)
	if err != nil {
		return nil, fmt.Errorf("PrivateKey.Sign: %w", err)
	}
	return signature, nil
}
