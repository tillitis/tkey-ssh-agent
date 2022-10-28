// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package util

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func InputUSS() ([]byte, error) {
	fmt.Printf("Enter phrase for the USS: ")
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("ReadPassword: %w", err)
	}
	fmt.Printf("\nRepeat the phrase: ")
	ussAgain, err := term.ReadPassword(0)
	if err != nil {
		return nil, fmt.Errorf("ReadPassword: %w", err)
	}
	fmt.Printf("\n")
	if bytes.Compare(secret, ussAgain) != 0 {
		return nil, fmt.Errorf("phrases did not match")
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("no phrase entered")
	}
	return secret, nil
}

func ReadUSS(fileUSS string) ([]byte, error) {
	var secret []byte
	var err error
	if fileUSS == "-" {
		if secret, err = io.ReadAll(os.Stdin); err != nil {
			return nil, fmt.Errorf("ReadAll: %w", err)
		}
	} else if secret, err = os.ReadFile(fileUSS); err != nil {
		return nil, fmt.Errorf("ReadFile: %w", err)
	}
	return secret, nil
}
