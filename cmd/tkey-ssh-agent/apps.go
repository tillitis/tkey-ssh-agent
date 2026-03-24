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
			digest: AppDigest(appBinaryPreCastor),
		},
		{
			name:   "tkey-device-signer castor-alpha-1",
			digest: AppDigest(appBinaryCastor),
		},
	}

	return list
}

// GetApp returns an app compatible with the device. Compatibility is
// determined by the UDI product ID pid. For backwards compatibility
// the pre Castor app is returned if pid is unknown.
func GetApp(pid uint8) []byte {
	switch pid {
	case tkeyclient.UDIPIDEngSample:
		return appBinaryCastor
	case tkeyclient.UDIPIDAcrab:
		return appBinaryPreCastor
	case tkeyclient.UDIPIDBellatrix:
		return appBinaryPreCastor
	case tkeyclient.UDIPIDBellatrixUnlocked:
		return appBinaryPreCastor
	case tkeyclient.UDIPIDCastor:
		return appBinaryCastor
	}

	return appBinaryPreCastor
}

func AppDigest(bin []byte) string {
	digest := sha512.Sum512(bin)
	return hex.EncodeToString(digest[:])
}
