/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// buildWithIndex returns a build output dir containing a CouchDB index under
// META-INF/statedb, mimicking what the build phase stages.
func buildWithIndex(g *WithT) string {
	dir, err := os.MkdirTemp("", "binarybuilder-bld")
	g.Expect(err).NotTo(HaveOccurred())
	indexDir := filepath.Join(dir, "META-INF", "statedb", "couchdb", "indexes")
	g.Expect(os.MkdirAll(indexDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(indexDir, "indexOwner.json"), []byte(`{"index":{"fields":["owner"]}}`), 0o644)).To(Succeed())
	return dir
}

func TestReleaseCopiesStatedbArtifacts(t *testing.T) {
	g := NewWithT(t)

	bld := buildWithIndex(g)
	rel, err := os.MkdirTemp("", "binarybuilder-rel")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"release", bld, rel})).To(Succeed())

	// The peer prefixes release dir contents with META-INF/, so the index must
	// land at <release>/statedb/couchdb/indexes/indexOwner.json.
	copied := filepath.Join(rel, "statedb", "couchdb", "indexes", "indexOwner.json")
	g.Expect(copied).To(BeAnExistingFile())
	contents, err := os.ReadFile(copied)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(contents)).To(ContainSubstring("owner"))
}

func TestReleaseNoStatedbIsNoOp(t *testing.T) {
	g := NewWithT(t)

	bld, err := os.MkdirTemp("", "binarybuilder-bld-empty")
	g.Expect(err).NotTo(HaveOccurred())
	rel, err := os.MkdirTemp("", "binarybuilder-rel")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(run([]string{"release", bld, rel})).To(Succeed())
	g.Expect(filepath.Join(rel, "statedb")).NotTo(BeAnExistingFile())
}

func TestReleaseTooFewArgs(t *testing.T) {
	g := NewWithT(t)
	g.Expect(run([]string{"release", "onlyone"})).NotTo(Succeed())
}
