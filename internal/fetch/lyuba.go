// Copyright 2026 Fernandes Samuel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fetch

// Lyuba packages (pkg.lyuba.dev). A directory that contains .lyu files is a
// Lyuba package: only its .lyu files count, the generated .go next to them
// (published for Go modules, see lyuba publish) is ignored. A module with no
// .lyu file at all is not a Lyuba module: it belongs on pkg.go.dev.

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/derrors"
	"golang.org/x/pkgsite/internal/godoc"
	"golang.org/x/pkgsite/internal/source"
	"golang.org/x/pkgsite/internal/stdlib"
)

// LyubaOnly makes the fetcher skip modules without .lyu files. It is enabled
// by the pkg.lyuba.dev commands (worker, frontend, pkgsite) and off by
// default, so that pkgsite's own tests, written against Go modules, stay valid.
var LyubaOnly = false

// ErrNotLyuba reports a module with no .lyu file. It is an exclusion (status
// 403, like the modules excluded by pkgsite), not an error: the worker sees
// every Go module of the index go by.
var ErrNotLyuba = fmt.Errorf("no .lyu file: not a Lyuba module (see pkg.go.dev): %w", derrors.Excluded)

// isSourceFile reports whether name is a file that makes up a package,
// Go or Lyuba.
func isSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".lyu")
}

// lyubaFiles returns the .lyu files of paths; empty if it is not a Lyuba
// package.
func lyubaFiles(paths []string) []string {
	var out []string
	for _, p := range paths {
		if strings.HasSuffix(p, ".lyu") {
			out = append(out, p)
		}
	}
	return out
}

func readLyuba(contentDir fs.FS, paths []string) ([]godoc.LyubaFile, error) {
	var files []godoc.LyubaFile
	for _, p := range paths {
		b, err := readFSFile(contentDir, p, MaxFileSize)
		if err != nil {
			return nil, err
		}
		files = append(files, godoc.LyubaFile{Name: path.Base(p), Src: string(b)})
	}
	return files, nil
}

func lyubaImportPath(modulePath, innerPath string) string {
	if modulePath == stdlib.ModulePath {
		return innerPath
	}
	return path.Join(modulePath, innerPath)
}

// loadLyubaPackageMeta returns the name and synopsis of a Lyuba package.
func loadLyubaPackageMeta(contentDir fs.FS, paths []string, innerPath string, modInfo *godoc.ModuleInfo) (*packageMeta, error) {
	files, err := readLyuba(contentDir, paths)
	if err != nil {
		return nil, err
	}
	d, err := godoc.ParseLyuba(files)
	if err != nil {
		return nil, &BadPackageError{Err: err}
	}
	return &packageMeta{path: lyubaImportPath(modInfo.ModulePath, innerPath), name: d.Name, synopsis: d.Synopsis}, nil
}

// loadLyubaPackage loads a Lyuba package, with a single documentation for
// all platforms (Lyuba has no per-platform build constraints for now).
func loadLyubaPackage(ctx context.Context, contentDir fs.FS, paths []string, innerPath string,
	sourceInfo *source.Info, modInfo *godoc.ModuleInfo) (*goPackage, error) {
	files, err := readLyuba(contentDir, paths)
	if err != nil {
		return nil, err
	}
	d, err := godoc.ParseLyuba(files)
	if err != nil {
		return nil, &BadPackageError{Err: err}
	}
	src, err := godoc.EncodeLyuba(files)
	if err != nil {
		return nil, err
	}
	decoded, err := godoc.DecodePackage(src)
	if err != nil {
		return nil, err
	}
	synopsis, imports, api, err := decoded.DocInfo(ctx, innerPath, sourceInfo, modInfo)
	if err != nil {
		return nil, fmt.Errorf("Lyuba doc: %w", err)
	}
	for _, s := range api {
		s.GOOS, s.GOARCH = internal.All, internal.All
	}
	importPath := lyubaImportPath(modInfo.ModulePath, innerPath)
	return &goPackage{
		path:    importPath,
		v1path:  internal.V1Path(importPath, modInfo.ModulePath),
		name:    d.Name,
		imports: imports,
		docs: []*internal.Documentation{{
			GOOS: internal.All, GOARCH: internal.All,
			Synopsis: synopsis, Source: src, API: api,
		}},
	}, nil
}
