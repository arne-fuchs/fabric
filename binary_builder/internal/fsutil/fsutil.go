/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package fsutil contains small filesystem helpers shared by the binary
// builder commands. It intentionally avoids external dependencies, in keeping
// with the rest of the builder.
package fsutil

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// CopyTree recursively copies the directory tree rooted at src into dst,
// preserving file permission bits. Directories are created as needed. It is a
// no-op (returning nil) if src does not exist, which lets callers copy
// optional directories unconditionally.
func CopyTree(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.WithMessagef(err, "could not stat %s", src)
	}

	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if fi.IsDir() {
			if err := os.MkdirAll(target, dirMode(fi.Mode().Perm())); err != nil {
				return errors.WithMessagef(err, "could not create %s", target)
			}
			return nil
		}

		return copyFile(path, target, fi.Mode().Perm())
	})
}

// copyFile copies a single file from src to dst, creating dst with the given
// permission bits.
func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errors.WithMessagef(err, "could not create parent of %s", dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return errors.WithMessagef(err, "could not open %s", src)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return errors.WithMessagef(err, "could not create %s", dst)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return errors.WithMessagef(err, "could not copy %s to %s", src, dst)
	}

	return out.Close()
}

// dirMode ensures copied directories remain traversable and writable by the
// owner regardless of the source permission bits.
func dirMode(perm os.FileMode) os.FileMode {
	return perm | 0o700
}
