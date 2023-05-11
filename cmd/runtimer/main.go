// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/internal/util"
	"github.com/tillitis/tkeyclient"
)

var (
	cmdSetTimer     = appCmd{0x01, "cmdSetTimer", tkeyclient.CmdLen32}
	rspSetTimer     = appCmd{0x02, "rspSetTimer", tkeyclient.CmdLen4}
	cmdSetPrescaler = appCmd{0x03, "cmdSetPrescaler", tkeyclient.CmdLen32}
	rspSetPrescaler = appCmd{0x04, "rspSetPrescaler", tkeyclient.CmdLen4}
	cmdStartTimer   = appCmd{0x05, "cmdStartTimer", tkeyclient.CmdLen1}
	rspStartTimer   = appCmd{0x06, "rspStartTimer", tkeyclient.CmdLen4}
)

type appCmd struct {
	code   byte
	name   string
	cmdLen tkeyclient.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() tkeyclient.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() tkeyclient.Endpoint {
	return tkeyclient.DestApp
}

func (c appCmd) String() string {
	return c.name
}

type Timer struct {
	tk *tkeyclient.TillitisKey // A connection to a TKey
}

// New allocates a struct for communicating with the timer app running
// on the TKey. You're expected to pass an existing connection to it,
// so use it like this:
//
//	tk := tkeyclient.New()
//	err := tk.Connect(port)
//	timer := NewTimer(tk)
func NewTimer(tk *tkeyclient.TillitisKey) Timer {
	var timer Timer

	timer.tk = tk

	return timer
}

// setInt sets an int with the command cmd
func (t Timer) setInt(sendCmd appCmd, expectedReceiveCmd appCmd, i int) error {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(sendCmd, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// The integer
	tx[2] = byte(i)
	tx[3] = byte(i >> 8)
	tx[4] = byte(i >> 16)
	tx[5] = byte(i >> 24)
	tkeyclient.Dump("tx", tx)
	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(expectedReceiveCmd, id)
	tkeyclient.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return fmt.Errorf("Command BAD")
	}

	return nil
}

func (t Timer) SetTimer(timer int) error {
	return t.setInt(cmdSetTimer, rspSetTimer, timer)
}

func (t Timer) SetPrescaler(prescaler int) error {
	return t.setInt(cmdSetPrescaler, rspSetPrescaler, prescaler)
}

func (t Timer) StartTimer() error {
	id := 2
	tx, err := tkeyclient.NewFrameBuf(cmdStartTimer, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(rspStartTimer, id)
	tkeyclient.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tkeyclient.StatusOK {
		return fmt.Errorf("Command BAD")
	}

	return nil
}

// matching device clock at 18 MHz
const defaultPrescaler = 18_000_000

func main() {
	var devPath string
	var speed, timer, prescaler int
	var verbose, helpOnly bool
	pflag.CommandLine.SortFlags = false
	pflag.StringVar(&devPath, "port", "",
		"Set serial port device `PATH`. If this is not passed, auto-detection will be attempted.")
	pflag.IntVar(&speed, "speed", tkeyclient.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	pflag.BoolVar(&verbose, "verbose", false,
		"Enable verbose output.")
	pflag.IntVar(&timer, "timer", 1,
		fmt.Sprintf("Set timer `VALUE` (seconds if prescaler is %d).", defaultPrescaler))
	pflag.IntVar(&prescaler, "prescaler", defaultPrescaler,
		"Set prescaler.")
	pflag.BoolVar(&helpOnly, "help", false, "Output this help.")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n%s", os.Args[0],
			pflag.CommandLine.FlagUsagesWrapped(80))
	}
	pflag.Parse()

	if helpOnly {
		pflag.Usage()
		os.Exit(0)
	}

	if !verbose {
		tkeyclient.SilenceLogging()
	}

	if devPath == "" {
		var err error
		devPath, err = util.DetectSerialPort(true)
		if err != nil {
			os.Exit(1)
		}
	}

	tk := tkeyclient.New()
	fmt.Printf("Connecting to device on serial port %s ...\n", devPath)
	if err := tk.Connect(devPath, tkeyclient.WithSpeed(speed)); err != nil {
		fmt.Printf("Could not open %s: %v\n", devPath, err)
		os.Exit(1)
	}
	exit := func(code int) {
		if err := tk.Close(); err != nil {
			fmt.Printf("tk.Close: %v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	tm := NewTimer(tk)

	err := tm.SetTimer(timer)
	if err != nil {
		fmt.Printf("SetTimer: %v\n", err)
		exit(1)
	}

	err = tm.SetPrescaler(prescaler)
	if err != nil {
		fmt.Printf("SetPrescaler: %v\n", err)
		exit(1)
	}

	t0 := time.Now()

	err = tm.StartTimer()
	if err != nil {
		fmt.Printf("StartTimer: %v\n", err)
		exit(1)
	}

	elapsed := time.Since(t0)

	fmt.Printf("Timer expired after %v seconds\n", elapsed.Seconds())

	exit(0)
}

func handleSignals(action func(), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		for {
			<-ch
			action()
		}
	}()
}
