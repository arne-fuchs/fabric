/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"
)

// writeMetadata writes a metadata.json into a fresh metadata dir and returns it.
func writeMetadata(g *WithT, contents string) string {
	dir, err := os.MkdirTemp("", "binarybuilder-meta")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(contents), 0o600)).To(Succeed())
	return dir
}

// writeSourceBinary writes a fake executable named name into a fresh source dir.
func writeSourceBinary(g *WithT, name string) string {
	dir, err := os.MkdirTemp("", "binarybuilder-src")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/true\n"), 0o755)).To(Succeed())
	return dir
}

func TestRunCopiesBinary(t *testing.T) {
	g := NewWithT(t)
	current := runtime.GOOS + "/" + runtime.GOARCH

	src := writeSourceBinary(g, "chaincode")
	meta := writeMetadata(g, `{"type":"binary","label":"l","chaincodeData":{"platform":"`+current+`"}}`)
	out, err := os.MkdirTemp("", "binarybuilder-out")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"build", src, meta, out})).To(Succeed())

	info, err := os.Stat(filepath.Join(out, "chaincode"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Mode().Perm() & 0o100).NotTo(BeZero(), "binary should be executable")
}

func TestRunCustomBinaryName(t *testing.T) {
	g := NewWithT(t)

	src := writeSourceBinary(g, "mychaincode")
	meta := writeMetadata(g, `{"type":"binary","chaincodeData":{"binary":"mychaincode"}}`)
	out, err := os.MkdirTemp("", "binarybuilder-out")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"build", src, meta, out})).To(Succeed())
	g.Expect(filepath.Join(out, "chaincode")).To(BeAnExistingFile())
}

func TestRunPlatformMismatch(t *testing.T) {
	g := NewWithT(t)

	src := writeSourceBinary(g, "chaincode")
	meta := writeMetadata(g, `{"type":"binary","chaincodeData":{"platform":"plan9/foo"}}`)
	out, err := os.MkdirTemp("", "binarybuilder-out")
	g.Expect(err).NotTo(HaveOccurred())

	err = run([]string{"build", src, meta, out})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("plan9/foo"))
}

func TestRunMissingBinary(t *testing.T) {
	g := NewWithT(t)

	src, err := os.MkdirTemp("", "binarybuilder-src-empty")
	g.Expect(err).NotTo(HaveOccurred())
	meta := writeMetadata(g, `{"type":"binary"}`)
	out, err := os.MkdirTemp("", "binarybuilder-out")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"build", src, meta, out})).NotTo(Succeed())
}

func TestRunWrongType(t *testing.T) {
	g := NewWithT(t)

	src := writeSourceBinary(g, "chaincode")
	meta := writeMetadata(g, `{"type":"golang"}`)
	out, err := os.MkdirTemp("", "binarybuilder-out")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"build", src, meta, out})).NotTo(Succeed())
}
