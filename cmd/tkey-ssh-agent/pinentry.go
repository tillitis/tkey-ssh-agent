// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/twpayne/go-pinentry-minimal/pinentry"
)

func getSecret(udi string, pinentryProgram string) ([]byte, error) {
	// Displaying the Unique Device Identifier (UDI) so the user will
	// know which stick they have plugged in.
	desc := fmt.Sprintf("%s needs a User Supplied Secret\n"+
		"(USS) for your TKey with number:\n"+
		"%v", progname, udi)

	if runtime.GOOS == "darwin" && pinentryProgram == "" {
		pin, err := macOSPrompt(desc, progname)
		if err != nil {
			return nil, fmt.Errorf("macOS Prompt: %w", err)
		}
		return []byte(pin), nil
	}

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
	} else if runtime.GOOS == "windows" {
		found := findWindowsPinentry()
		if found != "" {
			le.Printf("Found gpgconf and got pinentry program: %s\n", found)
			opts = append(opts, pinentry.WithBinaryName(found))
		}
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

func findWindowsPinentry() string {
	// When Gpg4win is installed using winget, the path to gpgconf
	// (and other gpg programs) is added to PATH (it is something like
	// `C:\Program Files (x86)\GnuPG\bin`). Given that, we try to find
	// Gpg4win's pinentry program. Inspired by how gpg-agent does it
	// on Windows, see --pinentry-program on
	// https://www.gnupg.org/documentation/manuals/gnupg/Agent-Options.html

	var found string

	knownProg, err := exec.LookPath("gpgconf.exe")
	if err != nil {
		le.Printf("LookPath: %s\n", err)
		return ""
	}
	// Dropping final "gpgconf.exe"
	gpgDir := filepath.Dir(knownProg)
	// Drop final "bin" if present
	if filepath.Base(gpgDir) == "bin" {
		gpgDir = filepath.Dir(gpgDir)
	}

	relExes := []string{`..\Gpg4win\bin\pinentry.exe`, `..\Gpg4win\pinentry.exe`}
	for _, relExe := range relExes {
		candidate := filepath.Join(gpgDir, relExe)
		_, err = os.Stat(candidate)
		if err != nil {
			le.Printf("Tried %s got: %s\n", candidate, err)
			continue
		}
		found = candidate
		break
	}

	if found == "" {
		for _, exe := range []string{`pinentry.exe`, `pinentry-basic.exe`} {
			candidate, err := exec.LookPath(exe)
			if err != nil {
				le.Printf("LookPath: %s\n", err)
				continue
			}
			found = candidate
			break
		}
	}

	return found
}

var macOSScriptTemplate = template.Must(template.New("script").Parse(`
var app = Application.currentApplication()
app.includeStandardAdditions = true
app.displayDialog(
	"{{ .Message }}", {
    defaultAnswer: "",
	withTitle: "{{ .Title }}",
    buttons: ["Cancel", "OK"],
    defaultButton: "OK",
	cancelButton: "Cancel",
    hiddenAnswer: true,
})`))

func macOSPrompt(msg, title string) (string, error) {
	script := new(bytes.Buffer)
	if err := macOSScriptTemplate.Execute(script, map[string]interface{}{
		"Message": strings.ReplaceAll(msg, "\n", `\n`), "Title": title,
	}); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	c := exec.Command("osascript", "-s", "se", "-l", "JavaScript")
	c.Stdin = script
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute osascript: %w", err)
	}
	var x struct {
		TextReturned string `json:"textReturned"`
	}
	if err := json.Unmarshal(out, &x); err != nil {
		return "", fmt.Errorf("failed to parse osascript output: %w", err)
	}
	return x.TextReturned, nil
}
