/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hyperledger/fabric/binary_builder/internal/fsutil"
	"github.com/hyperledger/fabric/binary_builder/internal/metadata"
	"github.com/pkg/errors"
)

var logger = log.New(os.Stderr, "", 0)

func main() {
	logger.Println("::Build")

	if err := run(os.Args); err != nil {
		logger.Printf("::Error: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("::Build phase completed")
}

// run implements the build phase. The peer invokes:
//
//	build CHAINCODE_SOURCE_DIR CHAINCODE_METADATA_DIR BUILD_OUTPUT_DIR
//
// It validates the package type and target platform, then stages the prebuilt
// executable into the build output directory as "chaincode". The run phase
// does not receive the metadata directory, so everything it needs must be
// written here.
func run(args []string) error {
	if len(args) < 4 {
		return errors.New("incorrect number of arguments, expected: build SOURCE_DIR METADATA_DIR OUTPUT_DIR")
	}

	sourceDir, metadataDir, outputDir := args[1], args[2], args[3]

	md, err := metadata.Read(metadataDir)
	if err != nil {
		return err
	}

	if strings.ToLower(md.Type) != metadata.Type {
		return fmt.Errorf("chaincode type should be %s, it is %s", metadata.Type, md.Type)
	}

	if err := checkPlatform(md.ChaincodeData.Platform); err != nil {
		return err
	}

	src := filepath.Join(sourceDir, md.ChaincodeData.BinaryName())
	dst := filepath.Join(outputDir, metadata.DefaultBinaryName)
	if err := copyExecutable(src, dst); err != nil {
		return err
	}

	// Carry any state database artifacts (e.g. CouchDB index definitions under
	// META-INF/statedb) from the package source into the build output, so the
	// release phase can hand them to the peer. The run phase does not need
	// them. This is a no-op when the package ships no META-INF directory.
	if err := fsutil.CopyTree(filepath.Join(sourceDir, "META-INF"), filepath.Join(outputDir, "META-INF")); err != nil {
		return errors.WithMessage(err, "could not copy META-INF into build output")
	}

	return nil
}

// checkPlatform fails when the package declares a target platform that does
// not match the platform of this peer. An empty platform is treated as "any".
func checkPlatform(platform string) error {
	if platform == "" {
		return nil
	}
	current := runtime.GOOS + "/" + runtime.GOARCH
	if platform != current {
		return fmt.Errorf("package targets %s but this peer is %s", platform, current)
	}
	return nil
}

// copyExecutable copies the prebuilt binary from src to dst and marks it
// executable.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.WithMessagef(err, "could not open chaincode binary %s", src)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return errors.WithMessagef(err, "could not create %s", dst)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return errors.WithMessagef(err, "could not copy chaincode binary to %s", dst)
	}

	return nil
}
