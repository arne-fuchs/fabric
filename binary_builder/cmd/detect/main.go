/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hyperledger/fabric/binary_builder/internal/metadata"
	"github.com/pkg/errors"
)

var logger = log.New(os.Stderr, "", 0)

func main() {
	logger.Println("::Detect")

	if err := run(os.Args); err != nil {
		logger.Printf("::Error: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("::Type detected as %s", metadata.Type)
}

// run implements the detect phase. The peer invokes:
//
//	detect CHAINCODE_SOURCE_DIR CHAINCODE_METADATA_DIR
//
// Returning nil (exit 0) tells the peer this builder handles the package.
func run(args []string) error {
	if len(args) < 3 {
		return errors.New("too few arguments, expected: detect SOURCE_DIR METADATA_DIR")
	}

	metadataDir := args[2]

	md, err := metadata.Read(metadataDir)
	if err != nil {
		return err
	}

	if strings.ToLower(md.Type) != metadata.Type {
		return fmt.Errorf("chaincode type not supported: %s", md.Type)
	}

	return nil
}
