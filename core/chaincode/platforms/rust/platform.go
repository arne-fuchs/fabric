/*
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
*/

package rust

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms/util"
)

var logger = flogging.MustGetLogger("chaincode.platform.rust")

// Type is the chaincode language identifier for Rust, as written to a package's
// metadata.json "type" field and used as the key in the platform registries.
//
// The other languages derive their name from the ChaincodeSpec_Type protobuf
// enum (e.g. pb.ChaincodeSpec_GOLANG.String()), but that enum — defined in the
// external fabric-protos repository — has no RUST member. Adding one is an
// upstream change: add RUST = 5 to ChaincodeSpec.Type in chaincode.proto,
// regenerate fabric-protos-go-apiv2, and bump the dependency. Until that lands,
// this constant is the single source of truth; replace its uses (here and in
// core/container/dockercontroller) with pb.ChaincodeSpec_RUST.String() once the
// enum exists.
const Type = "RUST"

// Platform for chaincodes written in Rust (e.g. using the fabric-sdk-rust
// chaincode library). Rust chaincode compiles to a native binary, so the build
// flow mirrors the golang platform: the source is compiled in a Rust toolchain
// image and the resulting binary is run in the slim fabric-baseos runtime
// image.
type Platform struct{}

// Name returns the name of this platform.
func (p *Platform) Name() string {
	return Type
}

// Returns whether the given file or directory exists or not.
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// ValidatePath validates that the chaincode path points at a local Rust crate.
func (p *Platform) ValidatePath(rawPath string) error {
	path, err := url.Parse(rawPath)
	if err != nil || path == nil {
		return fmt.Errorf("invalid path: %s", err)
	}

	// Treat empty scheme as a local filesystem path.
	if path.Scheme != "" {
		return nil
	}

	pathToCheck, err := filepath.Abs(rawPath)
	if err != nil {
		return fmt.Errorf("error obtaining absolute path of the chaincode: %s", err)
	}

	exists, err := pathExists(pathToCheck)
	if err != nil {
		return fmt.Errorf("error validating chaincode path: %s", err)
	}
	if !exists {
		return fmt.Errorf("path to chaincode does not exist: %s", rawPath)
	}

	// A Rust crate is rooted at a Cargo.toml manifest.
	manifestExists, err := pathExists(filepath.Join(pathToCheck, "Cargo.toml"))
	if err != nil {
		return fmt.Errorf("error validating chaincode path: %s", err)
	}
	if !manifestExists {
		return fmt.Errorf("no Cargo.toml found at the root of the chaincode path: %s", rawPath)
	}

	return nil
}

func (p *Platform) ValidateCodePackage(code []byte) error {
	// Scan the provided tarball to ensure it only contains source code under the
	// src folder (the crate) or META-INF (state database artifacts). As with the
	// other platforms, the build container remains the last line of defense; this
	// just keeps obvious garbage out of the system.
	re := regexp.MustCompile(`^(/)?(src|META-INF)/.*`)
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	foundCargoToml := false
	for {
		header, err := tr.Next()
		if err != nil {
			// We only get here if there are no more entries to scan.
			break
		}

		// Check name for conforming path.
		if !re.MatchString(header.Name) {
			return fmt.Errorf("illegal file detected in payload: \"%s\"", header.Name)
		}
		if header.Name == "src/Cargo.toml" {
			foundCargoToml = true
		}

		// Check that file mode makes sense. Acceptable flags:
		//      ISREG      == 0100000
		//      -rw-rw-rw- == 0666
		// Anything else is suspect in this context and will be rejected.
		if header.Mode&^0o100666 != 0 {
			return fmt.Errorf("illegal file mode detected for file %s: %o", header.Name, header.Mode)
		}
	}
	if !foundCargoToml {
		return fmt.Errorf("no Cargo.toml found at the root of the chaincode package")
	}

	return nil
}

// GetDeploymentPayload packages the crate source into src/$file entries in
// .tar.gz format. The compiled target directory is excluded; the binary is
// produced later in the build container.
func (p *Platform) GetDeploymentPayload(path string) ([]byte, error) {
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	folder := path
	if folder == "" {
		return nil, errors.New("ChaincodeSpec's path cannot be empty")
	}

	// Trim trailing slash if it exists.
	if folder[len(folder)-1] == '/' {
		folder = folder[:len(folder)-1]
	}

	logger.Debugf("Packaging Rust crate from path %s", folder)

	if err := util.WriteFolderToTarPackage(tw, folder, []string{"target"}, nil, nil); err != nil {
		logger.Errorf("Error writing folder to tar package %s", err)
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

	tw.Close()
	gw.Close()

	return payload.Bytes(), nil
}

func (p *Platform) GenerateDockerfile() (string, error) {
	var buf []string

	buf = append(buf, "FROM "+util.GetDockerImageFromConfig("chaincode.rust.runtime"))
	buf = append(buf, "ADD binpackage.tar /usr/local/bin")

	return strings.Join(buf, "\n"), nil
}

// rustTargets maps a Go architecture (runtime.GOARCH) to the Rust target triple
// the chaincode is compiled for. The chaincode is built natively in a Rust
// image of the peer's own architecture, so only the host triples are needed —
// no cross-compilation toolchain is required in the build image.
var rustTargets = map[string]string{
	"amd64": "x86_64-unknown-linux-gnu",
	"arm64": "aarch64-unknown-linux-gnu",
}

// buildScriptTemplate compiles the crate in release mode and copies the
// resulting binary to the well-known output location the runtime image
// launches. The crate's binary is named after the package (or its [[bin]]
// entry), so the first executable produced under target/<triple>/release is
// selected.
//
// crt-static statically links the C runtime so the binary has no external
// dependency on the build image; a committed Cargo.lock plus --locked keeps the
// build deterministic. %[1]s is the Rust target triple.
const buildScriptTemplate = `
set -e
cd /chaincode/input/src
RUSTFLAGS="-C target-feature=+crt-static" cargo build --release --locked --target %[1]s
bin="$(find target/%[1]s/release -maxdepth 1 -type f -perm -u+x ! -name '*.d' | head -n 1)"
if [ -z "$bin" ]; then
	echo "no executable produced by cargo build" >&2
	exit 1
fi
cp "$bin" /chaincode/output/chaincode
echo Done!
`

// buildScriptForArch returns the build script for the given Go architecture, or
// an error if Rust chaincode is not supported on that architecture.
func buildScriptForArch(goarch string) (string, error) {
	triple, ok := rustTargets[goarch]
	if !ok {
		return "", fmt.Errorf("unsupported architecture for rust chaincode: %s", goarch)
	}
	return fmt.Sprintf(buildScriptTemplate, triple), nil
}

func (p *Platform) DockerBuildOptions(path string, goVer, osVer, archVer string) (util.DockerBuildOptions, error) {
	script, err := buildScriptForArch(runtime.GOARCH)
	if err != nil {
		return util.DockerBuildOptions{}, err
	}

	env := []string{}
	for _, key := range []string{"CARGO_HTTP_PROXY", "CARGO_HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	return util.DockerBuildOptions{
		// Unlike golang (which builds in the default fabric-ccenv builder), Rust
		// needs the cargo toolchain, so we point at a dedicated build image.
		Image: util.GetDockerImageFromConfig("chaincode.rust.builder"),
		Cmd:   script,
		Env:   env,
	}, nil
}
