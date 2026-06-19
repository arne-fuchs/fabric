/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rust

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var platform = &Platform{}

type packageFile struct {
	packagePath string
	mode        int64
}

func TestName(t *testing.T) {
	require.Equal(t, "RUST", platform.Name())
}

func TestValidatePath(t *testing.T) {
	err := platform.ValidatePath("there/is/no/way/this/path/exists")
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "path to chaincode does not exist"),
		"unexpected error: %v", err)

	err = platform.ValidatePath("http://something bad/because/it/has/the/space")
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "invalid path"),
		"unexpected error: %v", err)

	// A directory without a Cargo.toml is not a Rust crate.
	dir := t.TempDir()
	err = platform.ValidatePath(dir)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "no Cargo.toml found"),
		"unexpected error: %v", err)

	// testdata is a minimal, valid crate.
	require.NoError(t, platform.ValidatePath("testdata"))
}

func TestValidateCodePackage(t *testing.T) {
	err := platform.ValidateCodePackage([]byte("dummy CodePackage content"))
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "failure opening codepackage gzip stream"),
		"unexpected error: %v", err)

	cp, err := makeCodePackage([]*packageFile{{"filename.txt", 0o100744}})
	require.NoError(t, err)
	err = platform.ValidateCodePackage(cp)
	require.Error(t, err, "file in archive root instead of src/ should be rejected")
	require.True(t, strings.HasPrefix(err.Error(), "illegal file detected in payload"),
		"unexpected error: %v", err)

	cp, err = makeCodePackage([]*packageFile{{"src/main.rs", 0o100744}})
	require.NoError(t, err)
	err = platform.ValidateCodePackage(cp)
	require.Error(t, err, "executable source file should be rejected")
	require.True(t, strings.HasPrefix(err.Error(), "illegal file mode detected for file"),
		"unexpected error: %v", err)

	cp, err = makeCodePackage([]*packageFile{{"src/main.rs", 0o100666}})
	require.NoError(t, err)
	err = platform.ValidateCodePackage(cp)
	require.Error(t, err, "package without Cargo.toml should be rejected")
	require.True(t, strings.HasPrefix(err.Error(), "no Cargo.toml found at the root of the chaincode package"),
		"unexpected error: %v", err)

	cp, err = makeCodePackage([]*packageFile{{"src/Cargo.toml", 0o100666}, {"META-INF/path/to/meta", 0o100744}})
	require.NoError(t, err)
	err = platform.ValidateCodePackage(cp)
	require.Error(t, err, "executable META-INF file should be rejected")
	require.True(t, strings.HasPrefix(err.Error(), "illegal file mode detected for file"),
		"unexpected error: %v", err)

	cp, err = makeCodePackage([]*packageFile{{"src/Cargo.toml", 0o100666}, {"src/src/main.rs", 0o100666}, {"META-INF/path/to/meta", 0o100666}})
	require.NoError(t, err)
	require.NoError(t, platform.ValidateCodePackage(cp))
}

func TestGetDeploymentPayload(t *testing.T) {
	_, err := platform.GetDeploymentPayload("")
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "ChaincodeSpec's path cannot be empty"),
		"unexpected error: %v", err)

	payload, err := platform.GetDeploymentPayload("testdata")
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	names := tarEntries(t, payload)
	require.Contains(t, names, "src/Cargo.toml")
	require.Contains(t, names, "src/Cargo.lock")
	require.Contains(t, names, "src/src/main.rs")
}

func TestGetDeploymentPayloadExcludesTarget(t *testing.T) {
	// The real testdata crate ships no compiled output, so build a copy with a
	// target/ directory to confirm it is excluded from the package.
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "target", "release"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname=\"x\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte("fn main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target", "release", "x"), []byte("binary"), 0o755))

	payload, err := platform.GetDeploymentPayload(dir)
	require.NoError(t, err)

	names := tarEntries(t, payload)
	require.Contains(t, names, "src/Cargo.toml")
	require.NotContains(t, names, "src/target/release/x", "target/ should be excluded from the package")
}

func TestGenerateDockerfile(t *testing.T) {
	str, err := platform.GenerateDockerfile()
	require.NoError(t, err)
	require.Contains(t, str, "/fabric-baseos:", "should build from the fabric-baseos runtime image")
	require.Contains(t, str, "ADD binpackage.tar /usr/local/bin")
}

func TestGenerateBuildOptions(t *testing.T) {
	opts, err := platform.DockerBuildOptions("pathname", "", "", "")
	require.NoError(t, err)

	require.Equal(t, "rust:1.98.0-bookworm", opts.Image)

	// The build script targets the host architecture.
	expected, err := buildScriptForArch(runtime.GOARCH)
	require.NoError(t, err)
	require.Equal(t, expected, opts.Cmd)
}

func TestBuildScriptForArch(t *testing.T) {
	amd64, err := buildScriptForArch("amd64")
	require.NoError(t, err)
	require.Contains(t, amd64, "--target x86_64-unknown-linux-gnu")
	require.Contains(t, amd64, "target/x86_64-unknown-linux-gnu/release")
	require.Contains(t, amd64, "target-feature=+crt-static")

	arm64, err := buildScriptForArch("arm64")
	require.NoError(t, err)
	require.Contains(t, arm64, "--target aarch64-unknown-linux-gnu")
	require.Contains(t, arm64, "target/aarch64-unknown-linux-gnu/release")

	_, err = buildScriptForArch("riscv64")
	require.ErrorContains(t, err, "unsupported architecture for rust chaincode: riscv64")
}

func tarEntries(t *testing.T, payload []byte) []string {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	require.NoError(t, err)
	tr := tar.NewReader(gr)

	var names []string
	for {
		header, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, header.Name)
	}
	return names
}

func makeCodePackage(pfiles []*packageFile) ([]byte, error) {
	contents := []byte("fake file's content")

	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	for _, f := range pfiles {
		if err := tw.WriteHeader(&tar.Header{
			Name: f.packagePath,
			Mode: f.mode,
			Size: int64(len(contents)),
		}); err != nil {
			return nil, fmt.Errorf("Error write header: %s", err)
		}

		if _, err := tw.Write(contents); err != nil {
			return nil, fmt.Errorf("Error writing contents: %s", err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

	gw.Close()

	return payload.Bytes(), nil
}

func TestMain(m *testing.M) {
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("could not read config %s\n", err)
		os.Exit(-1)
	}
	os.Exit(m.Run())
}
