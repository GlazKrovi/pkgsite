// Copyright 2026 Fernandes Samuel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fetch

// Packages Lyuba (pkg.lyuba.dev). Un dossier qui contient des .lyu est un
// package Lyuba : seuls ses .lyu comptent, le .go généré à côté (publié
// pour les modules go, cf lyuba publish) est ignoré. Un module sans aucun
// .lyu n'est pas un module Lyuba : il reste sur pkg.go.dev.

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

// LyubaOnly : ignorer les modules sans .lyu. Activé par les commandes de
// pkg.lyuba.dev (worker, frontend, pkgsite) ; faux par défaut, pour que
// les tests de pkgsite, écrits sur des modules go, restent valables.
var LyubaOnly = false

// ErrNotLyuba : le module ne contient aucun .lyu. C'est une exclusion
// (statut 403, comme les modules exclus de pkgsite), pas une erreur : le
// worker voit passer tous les modules go de l'index.
var ErrNotLyuba = fmt.Errorf("aucun fichier .lyu : pas un module Lyuba (voir pkg.go.dev) : %w", derrors.Excluded)

// isSourceFile : les fichiers qui font un package, go ou Lyuba.
func isSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".lyu")
}

// lyubaFiles : les .lyu de la liste ; vide si ce n'est pas un package Lyuba.
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

// loadLyubaPackageMeta : nom et synopsis d'un package Lyuba.
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

// loadLyubaPackage : un package Lyuba, une seule documentation pour toutes
// les plateformes (pas de contraintes de build par plateforme en Lyuba
// pour l'instant).
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
		return nil, fmt.Errorf("doc Lyuba : %w", err)
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
