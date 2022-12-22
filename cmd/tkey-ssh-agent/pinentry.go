// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"

	"github.com/twpayne/go-pinentry-minimal/pinentry"
)

func getSecret(udi string, pinentryProgram string) ([]byte, error) {
	// Displaying the Unique Device Identifier (UDI) so the user will
	// know which stick they have plugged in.
	desc := fmt.Sprintf("%s needs a User Supplied Secret\n"+
		"(USS) for your TKey with number:\n"+
		"%v", progname, udi)

	// The default pinentry program (binaryName) in the client is
	// "pinentry".
	opts := []pinentry.ClientOption{
		// Try to get pinentry program from gpg-agent.conf
		pinentry.WithBinaryNameFromGnuPGAgentConf(),
		pinentry.WithGPGTTY(),
		pinentry.WithDesc(desc),
		// pinentry-gnome3 uses Prompt as a title so we don't use the
		// USS abbreviation, and skip trailing ":".
		pinentry.WithPrompt("User Supplied Secret"),
		// Title is not displayed by all pinentry programs (or
		// displayed obscurely in window title).
		pinentry.WithTitle(progname),
	}

	// If argument is passed, add option to override the pinentry program
	if pinentryProgram != "" {
		opts = append(opts, pinentry.WithBinaryName(pinentryProgram))
	}

	client, err := pinentry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("pinentry.NewClient: %w", err)
	}

	defer client.Close()

	pin, _, err := client.GetPIN()
	if err != nil {
		return nil, fmt.Errorf("pinentry GetPin: %w", err)
	}
	return []byte(pin), nil
}
