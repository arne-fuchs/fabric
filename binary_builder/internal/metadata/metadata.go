/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package metadata contains the chaincode package metadata structures shared
// by the binary builder's detect, build and run commands.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Type is the chaincode package type handled by the binary builder.
const Type = "binary"

// DefaultBinaryName is the name of the executable looked for inside the
// chaincode package (and the name it is staged as in the build output) when
// ChaincodeData.Binary is not set.
const DefaultBinaryName = "chaincode"

// ChaincodeMetadata represents the metadata.json file supplied in the
// chaincode package. Only the fields relevant to the binary builder are
// modelled; unknown fields are ignored.
type ChaincodeMetadata struct {
	Type          string        `json:"type"`
	Label         string        `json:"label"`
	ChaincodeData ChaincodeData `json:"chaincodeData"`
}

// ChaincodeData holds the binary-builder specific configuration, nested under
// the "chaincodeData" key of metadata.json to avoid clashing with keys owned
// by the external builders and launchers feature itself.
type ChaincodeData struct {
	// Binary is the name of the executable inside code.tar.gz. Optional;
	// defaults to DefaultBinaryName.
	Binary string `json:"binary"`
	// Platform, when set, is the "<goos>/<goarch>" the package targets. The
	// build phase fails with a clear error if it does not match the peer.
	Platform string `json:"platform"`
}

// BinaryName returns the configured binary name, or DefaultBinaryName when
// unset.
func (d ChaincodeData) BinaryName() string {
	if d.Binary == "" {
		return DefaultBinaryName
	}
	return d.Binary
}

// Read reads and parses metadata.json from the supplied metadata directory.
func Read(metadataDir string) (*ChaincodeMetadata, error) {
	metadataFile := filepath.Clean(filepath.Join(metadataDir, "metadata.json"))

	contents, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, errors.WithMessagef(err, "%s not readable", metadataFile)
	}

	var md ChaincodeMetadata
	if err := json.Unmarshal(contents, &md); err != nil {
		return nil, errors.WithMessagef(err, "unable to parse %s", metadataFile)
	}

	return &md, nil
}
