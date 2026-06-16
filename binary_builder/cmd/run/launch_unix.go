//go:build !windows
// +build !windows

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"syscall"

	"github.com/pkg/errors"
)

// launch replaces the current process image with the chaincode executable so
// that signals delivered by the peer's external builder framework (e.g.
// SIGTERM on stop) reach the chaincode directly. This mirrors `exec` in a
// shell run script.
func launch(plan launchPlan) error {
	if err := syscall.Exec(plan.Path, plan.Args, plan.Env); err != nil {
		return errors.WithMessagef(err, "could not exec chaincode %s", plan.Path)
	}
	return nil // unreachable on success: the process image has been replaced
}
