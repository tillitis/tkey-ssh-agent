// Copyright (C) 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

//go:build unix

package util

import (
	"fmt"
	"os"

	"github.com/gen2brain/beeep"
)

func Notify(progname, msg string) {
	// Using progname as title
	if err := beeep.Notify(progname, msg, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Notify message %q failed: %s\n", msg, err)
	}
}
