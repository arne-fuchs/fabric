/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric/binary_builder/internal/fsutil"
	"github.com/pkg/errors"
)

var logger = log.New(os.Stderr, "", 0)

func main() {
	logger.Println("::Release")

	if err := run(os.Args); err != nil {
		logger.Printf("::Error: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("::Release phase completed")
}

// run implements the release phase. The peer invokes:
//
//	release BUILD_OUTPUT_DIR RELEASE_OUTPUT_DIR
//
// It hands the peer any state database artifacts staged by the build phase by
// copying META-INF/statedb (e.g. CouchDB index definitions) into the release
// directory under statedb/. The peer collects the release directory and
// deploys the indexes to the state database. With LevelDB, or when the package
// ships no statedb artifacts, there is nothing to copy and release is a no-op.
func run(args []string) error {
	if len(args) < 3 {
		return errors.New("incorrect number of arguments, expected: release BUILD_OUTPUT_DIR RELEASE_OUTPUT_DIR")
	}

	buildOutputDir, releaseDir := args[1], args[2]

	statedbSrc := filepath.Join(buildOutputDir, "META-INF", "statedb")
	statedbDst := filepath.Join(releaseDir, "statedb")
	if err := fsutil.CopyTree(statedbSrc, statedbDst); err != nil {
		return errors.WithMessage(err, "could not copy statedb artifacts into release output")
	}

	return nil
}
