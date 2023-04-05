// Copyright (C) 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

//go:build windows

package util

import (
	"fmt"
	"os"

	"github.com/gen2brain/beeep"
	"github.com/go-toast/toast"
	"golang.org/x/sys/windows"
)

var isWindows10 bool

func init() {
	maj, _, _ := windows.RtlGetNtVersionNumbers()
	isWindows10 = (maj >= 10)
}

func Notify(progname, msg string) {
	// Doing this because beeep doesn't let us set appID
	if isWindows10 {
		// Skipping msg title in win10+ toast. AppID (progname) will
		// be displayed at the top of the toast frame.
		notification := toast.Notification{
			AppID:   progname,
			Title:   "",
			Message: msg,
			Icon:    "",
		}
		if err := notification.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "toastNotify message %q failed: %s\n", msg, err)
		}
		return
	}

	// Using progname as title
	if err := beeep.Notify(progname, msg, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Notify message %q failed: %s\n", msg, err)
	}
}
