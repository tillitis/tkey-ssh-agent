// SPDX-FileCopyrightText: 2025 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"fmt"

	"github.com/tillitis/tkeyclient"
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
	AppTypeUnknown = iota
	AppTypePreCastor
	AppTypeCastor
)

type EmbeddedApp struct {
	name   string
	digest string
	app    []byte
}

type Apps struct {
	appMap map[AppType]EmbeddedApp
}

// NewDeviceApps returns type Apps which has methods to get at a
// mapping between a specific TKey UDI and the app type to use.
//
// Different app types are needed depending on the TKey platform used.
// Currently there are two types: AppTypePreCastor (Acrab, Bellatrix
// models of TKey) and AppTypeCastor (the Castor model).
//
// The mapping between the app type to use is usually done by looking
// at the UDI product ID. See GetApp().
func NewDeviceApps() Apps {
	var apps Apps

	apps.appMap = map[AppType]EmbeddedApp{
		AppTypePreCastor: {
			name:   "tkey-device-signer 1.0.2",
			digest: embeddedAppDigest(appBinaryV1_0_2),
			app:    appBinaryV1_0_2,
		},
		AppTypeCastor: {
			name:   "tkey-device-signer castor-alpha-1",
			digest: embeddedAppDigest(appBinaryVCastorAlpha1),
			app:    appBinaryVCastorAlpha1,
		},
	}

	return apps
}

// Map returns the map between the app type and the embedded device
// app.
func (a Apps) Map() map[AppType]EmbeddedApp {
	return a.appMap
}

// GetApp looks up what type of app is needed depending on the UDI
// product ID. It returns the app and any error.
func (a Apps) GetApp(udi tkeyclient.UDI) ([]byte, error) {
	t := identifyAppType(udi)
	if t == AppTypeUnknown {
		return nil, fmt.Errorf("unknown device")
	}

	return a.appMap[t].app, nil
}

func embeddedAppDigest(bin []byte) string {
	digest := sha512.Sum512(bin)
	return hex.EncodeToString(digest[:])
}

func identifyAppType(udi tkeyclient.UDI) AppType {
	// XXX product ID 0 is assumed to be Castor-compatible.
	if udi.ProductID == tkeyclient.UDIPIDCastor || udi.ProductID == 0 {
		return AppTypeCastor
	} else if udi.ProductID >= tkeyclient.UDIPIDAcrab && udi.ProductID <= tkeyclient.UDIPIDBellatrix {
		return AppTypePreCastor
	}

	return AppTypeUnknown
}
