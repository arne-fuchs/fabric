/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric/binary_builder/internal/metadata"
	"github.com/pkg/errors"
)

var logger = log.New(os.Stderr, "", 0)

// chaincodeJSON models launchDir/chaincode.json, written by the peer's
// external builder framework (see core/container/externalbuilder runConfig).
type chaincodeJSON struct {
	ChaincodeID string `json:"chaincode_id"`
	PeerAddress string `json:"peer_address"`
	ClientCert  string `json:"client_cert"`
	ClientKey   string `json:"client_key"`
	RootCert    string `json:"root_cert"`
	MSPID       string `json:"mspid"`
}

// launchPlan is the fully-resolved description of how to start the chaincode.
// It is produced by buildLaunchPlan (pure, testable) and consumed by launch
// (platform specific).
type launchPlan struct {
	Path  string            // absolute path to the executable
	Args  []string          // argv, including Args[0]
	Env   []string          // complete environment
	Files map[string][]byte // files to write before launching (path -> contents)
}

func main() {
	logger.Println("::Run")

	if err := run(os.Args, os.Environ()); err != nil {
		logger.Printf("::Error: %v\n", err)
		os.Exit(1)
	}
}

// run implements the run phase. The peer invokes:
//
//	run BUILD_OUTPUT_DIR RUN_METADATA_DIR
//
// BUILD_OUTPUT_DIR holds the staged "chaincode" executable; RUN_METADATA_DIR
// holds chaincode.json with the peer connection details. The metadata.json
// file is NOT available at this phase.
func run(args, baseEnv []string) error {
	if len(args) < 3 {
		return errors.New("incorrect number of arguments, expected: run OUTPUT_DIR LAUNCH_DIR")
	}

	outputDir, launchDir := args[1], args[2]

	raw, err := os.ReadFile(filepath.Join(launchDir, "chaincode.json"))
	if err != nil {
		return errors.WithMessage(err, "could not read chaincode.json")
	}

	binaryPath := filepath.Join(outputDir, metadata.DefaultBinaryName)
	plan, err := buildLaunchPlan(binaryPath, launchDir, raw, baseEnv)
	if err != nil {
		return err
	}

	for path, contents := range plan.Files {
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			return errors.WithMessagef(err, "could not write %s", path)
		}
	}

	return launch(plan)
}

// buildLaunchPlan translates chaincode.json into the executable invocation.
//
// Two conventions are populated so both purpose-built and stock chaincode work:
//   - METADATA: the raw chaincode.json, read by chaincode that parses it directly.
//   - CORE_* environment variables and on-disk TLS material, the standard
//     fabric chaincode shim contract, so unmodified third-party binaries run.
func buildLaunchPlan(binaryPath, launchDir string, raw []byte, baseEnv []string) (launchPlan, error) {
	var cc chaincodeJSON
	if err := json.Unmarshal(raw, &cc); err != nil {
		return launchPlan{}, errors.WithMessage(err, "could not parse chaincode.json")
	}

	env := append([]string{}, baseEnv...)
	env = append(env,
		"METADATA="+string(raw),
		"CORE_CHAINCODE_ID_NAME="+cc.ChaincodeID,
		"CORE_PEER_LOCALMSPID="+cc.MSPID,
	)

	files := map[string][]byte{}
	if cc.ClientCert != "" {
		clientCert := filepath.Join(launchDir, "client.crt")
		clientKey := filepath.Join(launchDir, "client.key")
		rootCert := filepath.Join(launchDir, "root.crt")
		files[clientCert] = []byte(cc.ClientCert)
		files[clientKey] = []byte(cc.ClientKey)
		files[rootCert] = []byte(cc.RootCert)
		env = append(env,
			"CORE_PEER_TLS_ENABLED=true",
			"CORE_TLS_CLIENT_CERT_FILE="+clientCert,
			"CORE_TLS_CLIENT_KEY_FILE="+clientKey,
			"CORE_PEER_TLS_ROOTCERT_FILE="+rootCert,
		)
	} else {
		env = append(env, "CORE_PEER_TLS_ENABLED=false")
	}

	return launchPlan{
		Path:  binaryPath,
		Args:  []string{binaryPath, "-peer.address=" + cc.PeerAddress},
		Env:   env,
		Files: files,
	}, nil
}
