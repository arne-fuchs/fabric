//go:build windows
// +build windows

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/pkg/errors"
)

// launch starts the chaincode as a child process and forwards termination
// signals to it. Windows has no exec-replace equivalent, so we run the child
// and propagate its exit code.
func launch(plan launchPlan) error {
	cmd := exec.Command(plan.Path, plan.Args[1:]...)
	cmd.Env = plan.Env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return errors.WithMessagef(err, "could not start chaincode %s", plan.Path)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)
	go func() {
		<-sigs
		_ = cmd.Process.Kill()
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return errors.WithMessagef(err, "chaincode %s failed", plan.Path)
	}
	return nil
}
