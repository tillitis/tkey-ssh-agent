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
var appBinaryV1_0_2 []byte

// nolint:typecheck // Avoid lint error when the embedding file is missing.
//
//go:embed device-app/signer.bin-castor-alpha-1
var appBinaryVCastorAlpha1 []byte

type AppType int

const (
	AppTypePreCastor = iota
	AppTypeCastor
)

type EmbeddedApp struct {
	name   string
	digest string
	binary []byte
}

type Apps struct {
	appMap map[AppType]EmbeddedApp
	pidMap map[uint8]AppType
}

// NewDeviceApps returns type Apps.
//
// Different app types are needed depending on the TKey platform used.
// Currently there are two types:
//
// - AppTypePreCastor (Acrab, Bellatrix models of TKey)
//
// - AppTypeCastor (the Castor model).
//
// The mapping between app type and the app binary is kept in appMap.
// You need to update this if you are changing the version of the
// binary or adding new app types.
//
// The mapping between UDI Product ID and the app type to use is kept
// updated in the pidMap. You need to update this if you're adding
// support for other TKey Product IDs.
//
// Use GetApp(productId) to get the app binary to use.
//
// Use List() to get a list of embedded app binaries.
func NewDeviceApps() Apps {
	var apps Apps

	apps.appMap = map[AppType]EmbeddedApp{
		AppTypePreCastor: {
			name:   "tkey-device-signer 1.0.2",
			digest: embeddedAppDigest(appBinaryV1_0_2),
			binary: appBinaryV1_0_2,
		},

		AppTypeCastor: {
			name:   "tkey-device-signer castor-alpha-1",
			digest: embeddedAppDigest(appBinaryVCastorAlpha1),
			binary: appBinaryVCastorAlpha1,
		},
	}

	// Map what app type a specific Product ID should use
	apps.pidMap = map[uint8]AppType{
		// For development reasons the Engineering Sample is
		// assumed to run the latest app type, currently
		// Castor.
		tkeyclient.UDIPIDEngSample: AppTypeCastor,
		tkeyclient.UDIPIDAcrab:     AppTypePreCastor,
		tkeyclient.UDIPIDBellatrix: AppTypePreCastor,
		tkeyclient.UDIPIDCastor:    AppTypeCastor,
	}

	return apps
}

// List data about the embedded binaries.
func (a Apps) List() []EmbeddedApp {
	var list []EmbeddedApp

	for _, app := range a.appMap {
		list = append(list, app)
	}

	return list
}

// GetApp looks up what type of app is needed depending on the UDI
// product ID pid. It returns the app binary and any error.
func (a Apps) GetApp(pid uint8) ([]byte, error) {

	appType, ok := a.pidMap[pid]
	if !ok {
		return nil, ErrNotFound
	}

	return a.appMap[appType].binary, nil
}

func embeddedAppDigest(bin []byte) string {
	digest := sha512.Sum512(bin)
	return hex.EncodeToString(digest[:])
}
