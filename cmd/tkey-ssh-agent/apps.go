// SPDX-FileCopyrightText: 2025 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	_ "embed"
)

// Variable containing the signer app for Engineering Sample, Acrab,
// and Bellatrix models of TKey.

// nolint:typecheck // Avoid lint error when the embedding file is missing.
//
//go:embed device-app/signer.bin-v1.0.2
var appBinaryPreCastor []byte

// Variable containing the signer app for the Castor model of TKey.

// nolint:typecheck // Avoid lint error when the embedding file is missing.
//
//go:embed device-app/signer.bin-castor-alpha-1
var appBinaryCastor []byte

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

func NewDeviceApps() map[AppType]EmbeddedApp {
	apps := map[AppType]EmbeddedApp{
		AppTypePreCastor: {
			name:   "tkey-device-signer 1.0.2",
			digest: embeddedAppDigest(appBinaryPreCastor),
			app:    appBinaryPreCastor,
		},
		AppTypeCastor: {
			name:   "tkey-device-signer castor-alpha-1",
			digest: embeddedAppDigest(appBinaryCastor),
			app:    appBinaryCastor,
		},
	}

	return apps
}
