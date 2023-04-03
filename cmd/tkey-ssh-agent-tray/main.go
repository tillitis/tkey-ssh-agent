// Copyright (C) 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/apenwarr/fixconsole"
	"github.com/getlantern/systray"
	"github.com/tawesoft/golib/v2/dialog"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
)

var le = log.New(os.Stderr, "", 0)

const (
	progname = "tkey-ssh-agent-tray"
	// Expected to be found next to ourselves
	mainExe = "tkey-ssh-agent.exe"
)

var notify = func(msg string) {
	util.Notify(progname, msg)
}

func main() {
	if runtime.GOOS != "windows" {
		le.Printf("Only implemented for windows\n")
		os.Exit(1)
	}

	// We're not supposed to be run in a console , but if we still are
	// then try to get our output into it
	if err := fixconsole.FixConsoleIfNeeded(); err != nil {
		le.Printf("FixConsole: %d\n", err)
	}
	le = log.New(os.Stderr, "", 0)

	ourExePath, err := os.Executable()
	if err != nil {
		notify("Could not find our own executable")
		le.Printf("os.Executable: %d\n", err)
		os.Exit(1)
	}

	mainExePath := filepath.Join(filepath.Dir(ourExePath), mainExe)

	args := os.Args[1:]
	if !contains(args, "-a") && !contains(args, "--agent-path") {
		notify("To get tkey-ssh-agent started, the tray-program should be passed at least the -a argument to set the name of the listening pipe.")
		os.Exit(2)
	}

	fmt.Printf("Starting \"%s\" with args %v\n", mainExePath, args)

	cmd := exec.Command(mainExePath, args...)
	// mainExe is built as a "console binary" (without `-H
	// windowsgui`), so when run without a console, windows will open
	// up a console for it unless we do this:
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err = cmd.Start(); err != nil {
		notify(fmt.Sprintf("Could not start \"%s\":\n%s", mainExe, err))
		le.Printf("Failed to start: %s\n", err)
		os.Exit(1)
	}
	le.Printf("Started with PID: %d\n", cmd.Process.Pid)

	mainCmdLine := fmt.Sprintf("%s %s", mainExe, strings.Join(args, " "))
	go tray(mainCmdLine, func() {
		if err = cmd.Process.Kill(); err != nil {
			le.Printf("Failed to stop %s on Quit: %s\n", mainExe, err)
		}
		os.Exit(0)
	})

	state, err := cmd.Process.Wait()
	if err != nil {
		notify(fmt.Sprintf("Failed to wait for %s:\n%s", mainExe, err))
		le.Printf("Failed to wait for %s: %s\n", mainExe, err)
		os.Exit(1)
	}

	if !state.Success() {
		notify(fmt.Sprintf("%s stopped with code %d.\n%s will exit.",
			mainExe, state.ExitCode(), progname))
	}
	le.Printf("%s stopped with code: %d\n", mainExe, state.ExitCode())

	le.Printf("%s is exiting\n", progname)
	os.Exit(state.ExitCode())
}

//go:embed trayicon.ico
var trayIconICO []byte

func tray(mainCmdLine string, onExit func()) {
	onReady := func() {
		le.Printf("Added icon to system tray\n")
		systray.SetTemplateIcon(trayIconICO, trayIconICO)
		systray.SetTitle(mainCmdLine)   // only on linux, macos
		systray.SetTooltip(mainCmdLine) // only on macos, windows

		// no menuitem tooltip on windows
		about := systray.AddMenuItem("About", "")
		go func() {
			for range about.ClickedCh {
				_ = dialog.Info(fmt.Sprintf(`TKey SSH Agent
Copyright (C) Tillitis AB

Source code is licensed under
GNU General Public License v2.0 only
unless otherwise noted in the source code.

Source repository: https://github.com/tillitis/tillitis-key1-apps
Tillitis: https://www.tillitis.se

Running: %s`, mainCmdLine))
			}
		}()

		quit := systray.AddMenuItem("Quit", "")
		go func() {
			<-quit.ClickedCh
			le.Printf("Quit from trayicon menu\n")
			systray.Quit()
		}()
	}

	systray.Run(onReady, onExit)
}

func contains(ss []string, e string) bool {
	for _, s := range ss {
		if s == e {
			return true
		}
	}
	return false
}
