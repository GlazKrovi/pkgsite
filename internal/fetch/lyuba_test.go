// Copyright 2026 Fernandes Samuel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fetch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/godoc"
)

const formesLyu = `// Package formes : des formes géométriques.
package formes

// Carre : un carré, défini par son côté.
type Carre record {
    New({
        Cote: float, // le côté
        etiquette = "carré",
    })

    // Aire : la surface du carré.
    Aire(): float { return Cote * Cote }
}

type unite record {}
`

func lyubaModule() fstest.MapFS {
	return fstest.MapFS{
		"go.mod":               {Data: []byte("module exemple.dev/geo\n\ngo 1.27.0\n")},
		"formes/formes.lyu":    {Data: []byte(formesLyu)},
		"formes/formes.lyu.go": {Data: []byte("package formes\n\n// go généré, à ignorer\nfunc Ignore() {}\n")},
		"outils/outil.go":      {Data: []byte("package outils\n")},
	}
}

func TestLyubaModule(t *testing.T) {
	defer func(v bool) { LyubaOnly = v }(LyubaOnly)
	LyubaOnly = true
	ctx := context.Background()

	metas, modInfo, _, err := extractPackageMetas(ctx, "exemple.dev/geo", "v1.0.0", lyubaModule())
	if err != nil {
		t.Fatal(err)
	}
	// outils (du go seul, dans un module Lyuba) n'est pas un package du site
	if len(metas) != 1 || metas[0].name != "formes" || metas[0].synopsis != "Package formes : des formes géométriques." {
		t.Fatalf("%+v", metas)
	}

	pkg, pvs, err := extractPackage(ctx, "exemple.dev/geo", "exemple.dev/geo/formes", lyubaModule(), nil, nil, modInfo)
	if err != nil || pvs.Status != 200 {
		t.Fatal(err, pvs)
	}
	d := pkg.docs[0]
	if d.GOOS != internal.All || !strings.HasPrefix(string(d.Source), "LYU1") || strings.Contains(string(d.Source), "Ignore") {
		t.Fatalf("%+v", d)
	}
	var names []string
	for _, s := range d.API {
		names = append(names, s.Name)
		for _, c := range s.Children {
			names = append(names, c.Name)
		}
	}
	if strings.Join(names, " ") != "Carre Carre.Aire" {
		t.Error(names)
	}

	// rendu : ce que verra la page du package
	dp, err := godoc.DecodePackage(d.Source)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := dp.Render(ctx, "formes", nil, modInfo, nil, internal.BuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	body := parts.Body.String()
	for _, want := range []string{`id="Carre"`, "type Carre record {", "Cote: float, // le côté", `id="Carre.Aire"`, "Aire : la surface du carré.", "des formes géométriques"} {
		if !strings.Contains(body, want) {
			t.Errorf("sans %q :\n%s", want, body)
		}
	}
	if strings.Contains(body, "unite") {
		t.Error("type privé affiché")
	}
	if !strings.Contains(parts.Outline.String(), `href="#Carre.Aire"`) {
		t.Error(parts.Outline)
	}

	// un module sans .lyu n'est pas un module Lyuba
	_, _, _, err = extractPackageMetas(ctx, "exemple.dev/go", "v1.0.0", fstest.MapFS{"a/a.go": {Data: []byte("package a\n")}})
	if !errors.Is(err, ErrNotLyuba) {
		t.Error(err)
	}
}
