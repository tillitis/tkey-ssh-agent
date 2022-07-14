package main

import (
	"crypto"
	"crypto/ed25519"
	"io"
)

type HardwareSigner struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (h *HardwareSigner) init() error {
	var err error
	h.pub, h.priv, err = ed25519.GenerateKey(nil)
	return err
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

func (h *HardwareSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return h.priv.Sign(rand, digest, opts)
}
