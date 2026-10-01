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

const shapesLyu = `// Package shapes: geometric shapes.
package shapes

// Square: a square, defined by its side.
type Square record {
    New({
        Side: float, // the side
        label = "square",
    })

    // Area: the surface of the square.
    Area(): float { return Side * Side }
}

type unit record {}
`

func lyubaModule() fstest.MapFS {
	return fstest.MapFS{
		"go.mod":               {Data: []byte("module example.dev/geo\n\ngo 1.27.0\n")},
		"shapes/shapes.lyu":    {Data: []byte(shapesLyu)},
		"shapes/shapes.lyu.go": {Data: []byte("package shapes\n\n// generated go, to ignore\nfunc Ignore() {}\n")},
		"tools/tool.go":        {Data: []byte("package tools\n")},
	}
}

func TestLyubaModule(t *testing.T) {
	defer func(v bool) { LyubaOnly = v }(LyubaOnly)
	LyubaOnly = true
	ctx := context.Background()

	metas, modInfo, _, err := extractPackageMetas(ctx, "example.dev/geo", "v1.0.0", lyubaModule())
	if err != nil {
		t.Fatal(err)
	}
	// tools (plain Go in a Lyuba module) is not a package of the site
	if len(metas) != 1 || metas[0].name != "shapes" || metas[0].synopsis != "Package shapes: geometric shapes." {
		t.Fatalf("%+v", metas)
	}

	pkg, pvs, err := extractPackage(ctx, "example.dev/geo", "example.dev/geo/shapes", lyubaModule(), nil, nil, modInfo)
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
	if strings.Join(names, " ") != "Square Square.Area" {
		t.Error(names)
	}

	// rendering: what the package page will show
	dp, err := godoc.DecodePackage(d.Source)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := dp.Render(ctx, "shapes", nil, modInfo, nil, internal.BuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	body := parts.Body.String()
	for _, want := range []string{`id="Square"`, "type Square record {", "Side: float, // the side", `id="Square.Area"`, "Area: the surface of the square.", "geometric shapes"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "unit") {
		t.Error("private type displayed")
	}
	if !strings.Contains(parts.Outline.String(), `href="#Square.Area"`) {
		t.Error(parts.Outline)
	}

	// a module without .lyu files is not a Lyuba module
	_, _, _, err = extractPackageMetas(ctx, "example.dev/go", "v1.0.0", fstest.MapFS{"a/a.go": {Data: []byte("package a\n")}})
	if !errors.Is(err, ErrNotLyuba) {
		t.Error(err)
	}
}
