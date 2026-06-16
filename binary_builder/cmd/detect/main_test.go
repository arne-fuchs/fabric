/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{name: "valid binary type", args: []string{"detect", "src", "testdata/validtype"}, expectErr: false},
		{name: "wrong type", args: []string{"detect", "src", "testdata/wrongtype"}, expectErr: true},
		{name: "missing metadata dir", args: []string{"detect", "src", "testdata/doesnotexist"}, expectErr: true},
		{name: "too few arguments", args: []string{"detect", "src"}, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			err := run(tt.args)
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
