// SPDX-FileCopyrightText: 2025 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"crypto/sha512"
	_ "embed"
	"encoding/hex"

	"github.com/tillitis/tkeyclient"
)

type constError string

func (err constError) Error() string {
	return string(err)
}

const (
	ErrNotFound = constError("not found")
)

// nolint:typecheck // Avoid lint error when the embedding file is missing.
//
//go:embed device-app/signer.bin-v1.0.2
var appBinaryPreCastor []byte

// nolint:typecheck // Avoid lint error when the embedding file is missing.
//
//go:embed device-app/signer.bin-castor-alpha-1
var appBinaryCastor []byte

type EmbeddedApp struct {
	name   string
	digest string
}

// List data about the embedded binaries.
func ListApps() []EmbeddedApp {
	list := []EmbeddedApp{
		{
			name:   "tkey-device-signer 1.0.2",
			digest: embeddedAppDigest(appBinaryPreCastor),
		},
		{
			name:   "tkey-device-signer castor-alpha-1",
			digest: embeddedAppDigest(appBinaryCastor),
		},
	}

	return list
}

// GetApp looks up what type of app is needed depending on the UDI
// product ID pid. It returns the app binary and any error.
func GetApp(pid uint8) ([]byte, error) {
	switch pid {
	case tkeyclient.UDIPIDEngSample:
		return appBinaryCastor, nil
	case tkeyclient.UDIPIDAcrab:
		return appBinaryPreCastor, nil
	case tkeyclient.UDIPIDBellatrix:
		return appBinaryPreCastor, nil
	case tkeyclient.UDIPIDCastor:
		return appBinaryCastor, nil
	}

	return nil, ErrNotFound
}

func embeddedAppDigest(bin []byte) string {
	digest := sha512.Sum512(bin)
	return hex.EncodeToString(digest[:])
}
