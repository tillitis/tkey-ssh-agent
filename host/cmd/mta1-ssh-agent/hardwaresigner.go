package main

import (
	"crypto"
	"crypto/ed25519"
	"io"
)

type hardwareSigner struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (h *hardwareSigner) init() error {
	var err error
	h.pub, h.priv, err = ed25519.GenerateKey(nil)
	return err
}

// implementing crypto.Signer below

func (h *hardwareSigner) Public() crypto.PublicKey {
	return h.pub
}

func (h *hardwareSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return h.priv.Sign(rand, digest, opts)
}
