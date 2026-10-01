// Copyright 2026 Fernandes Samuel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godoc

// Documentation of Lyuba packages (pkg.lyuba.dev). A Lyuba package is stored
// under the "LYU1" encoding: its .lyu sources, as json. At render time the
// Lyuba compiler's parser reads them back, and lyuba/doc extracts what is
// exported.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/GlazKrovi/lyuba/ast"
	"github.com/GlazKrovi/lyuba/doc"
	"github.com/GlazKrovi/lyuba/parser"
	"github.com/google/safehtml/uncheckedconversions"
	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/godoc/dochtml"
	"golang.org/x/pkgsite/internal/source"
)

const lyubaEncodingType = "LYU1"

// LyubaFile is a .lyu file of a package.
type LyubaFile struct {
	Name string // relative to the package directory
	Src  string
}

// IsLyuba reports whether the package comes from .lyu files.
func (p *Package) IsLyuba() bool { return p.lyuba != nil }

// EncodeLyuba encodes the sources of a Lyuba package, to be stored as
// Documentation.Source.
func EncodeLyuba(files []LyubaFile) ([]byte, error) {
	b, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}
	return append([]byte(lyubaEncodingType), b...), nil
}

func decodeLyuba(data []byte) (*Package, error) {
	var files []LyubaFile
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}
	return &Package{lyuba: files}, nil
}

// ParseLyuba returns the documentation of a Lyuba package. Syntax errors do
// not prevent displaying it (the parser recovers after an error).
func ParseLyuba(files []LyubaFile) (*doc.Package, error) {
	var asts []*ast.File
	var errs []string
	for _, f := range files {
		a, perrs := parser.ParseFile(f.Name, f.Src)
		for _, e := range perrs {
			errs = append(errs, f.Name+":"+e.Error())
		}
		asts = append(asts, a)
	}
	d := doc.New(asts)
	if d.Name == "" {
		return nil, fmt.Errorf("no readable Lyuba package: %s", strings.Join(errs, " ; "))
	}
	return d, nil
}

// lyubaDocInfo returns the synopsis, imports and symbols (search, history).
func (p *Package) lyubaDocInfo() (string, []string, []*internal.Symbol, error) {
	d, err := ParseLyuba(p.lyuba)
	if err != nil {
		return "", nil, nil, err
	}
	var api []*internal.Symbol
	for _, c := range d.Consts {
		api = append(api, &internal.Symbol{SymbolMeta: internal.SymbolMeta{
			Name: c.Name, Synopsis: c.Decl, Section: internal.SymbolSectionConstants, Kind: internal.SymbolKindConstant}})
	}
	for _, t := range d.Types {
		s := &internal.Symbol{SymbolMeta: internal.SymbolMeta{
			Name: t.Name, Synopsis: "type " + t.Name + " " + t.Kind, Section: internal.SymbolSectionTypes, Kind: internal.SymbolKindType}}
		for _, c := range t.Consts {
			s.Children = append(s.Children, &internal.SymbolMeta{Name: t.Name + "." + c.Name, Synopsis: c.Decl,
				Section: internal.SymbolSectionTypes, Kind: internal.SymbolKindConstant, ParentName: t.Name})
		}
		for _, m := range t.Methods {
			s.Children = append(s.Children, &internal.SymbolMeta{Name: t.Name + "." + m.Name, Synopsis: firstLine(m.Decl),
				Section: internal.SymbolSectionTypes, Kind: internal.SymbolKindMethod, ParentName: t.Name})
		}
		api = append(api, s)
	}
	return d.Synopsis, d.Imports, api, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

// ===== html rendering, with pkgsite's css classes

type lyubaView struct {
	*doc.Package
	src func(file string, line int) string
}

func (v lyubaView) Src(file string, line int) string { return v.src(file, line) }

var lyubaFuncs = template.FuncMap{
	"para": renderDocText,
	"first": func(s string) string {
		return firstLine(s)
	},
}

// renderDocText renders a comment as paragraphs (blank line), with clickable
// http(s) links.
func renderDocText(s string) template.HTML {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, p := range regexp.MustCompile(`\n\s*\n`).Split(strings.TrimSpace(s), -1) {
		esc := template.HTMLEscapeString(p)
		esc = urlRE.ReplaceAllString(esc, `<a href="$0">$0</a>`)
		b.WriteString("<p>" + esc + "</p>\n")
	}
	return template.HTML(b.String())
}

var urlRE = regexp.MustCompile(`https?://[^\s<>"]+[^\s<>".,;:!?)]`)

var lyubaBody = template.Must(template.New("body").Funcs(lyubaFuncs).Parse(`
<div class="Documentation-content js-docContent">
{{- if .Doc}}
  <section class="Documentation-overview">
    <h3 tabindex="-1" id="pkg-overview" class="Documentation-overviewHeader">Overview <a href="#pkg-overview" aria-label="Go to Overview">¶</a></h3>
    {{para .Doc}}
  </section>
{{- end}}
{{- if or .Consts .Types}}
  <section class="Documentation-index">
    <h3 id="pkg-index" class="Documentation-indexHeader">Index <a href="#pkg-index" aria-label="Go to Index">¶</a></h3>
    <ul class="Documentation-indexList">
      {{- if .Consts}}<li class="Documentation-indexConstants"><a href="#pkg-constants">Constants</a></li>{{end}}
      {{- range .Types}}{{$t := .Name}}
      <li class="Documentation-indexType"><a href="#{{.Name}}">type {{.Name}} {{.Kind}}</a></li>
      {{- with .Methods}}<li><ul class="Documentation-indexTypeMethods">{{range .}}<li><a href="#{{$t}}.{{.Name}}">{{first .Decl}}</a></li>{{end}}</ul></li>{{end}}
      {{- end}}
    </ul>
  </section>
  <h3 tabindex="-1" id="pkg-constants" class="Documentation-constantsHeader">Constants <a href="#pkg-constants" aria-label="Go to Constants">¶</a></h3>
  <section class="Documentation-constants">
  {{- range .Consts}}
    <div class="Documentation-declaration" id="{{.Name}}">
      <span class="Documentation-declarationLink"><a href="{{$.Src .File .Line}}">View Source</a></span>
      <pre>{{.Decl}}</pre>
    </div>
    {{para .Doc}}
  {{- else}}
    <p class="Documentation-empty">No exported constants.</p>
  {{- end}}
  </section>
  <h3 tabindex="-1" id="pkg-types" class="Documentation-typesHeader">Types <a href="#pkg-types" aria-label="Go to Types">¶</a></h3>
  <section class="Documentation-types">
  {{- range .Types}}{{$t := .}}
    <div class="Documentation-type">
      <h4 tabindex="-1" id="{{.Name}}" data-kind="type" class="Documentation-typeHeader">
        <span>type <a class="Documentation-source" href="{{$.Src .File .Line}}">{{.Name}}</a> <span class="Documentation-kind">{{.Kind}}</span></span>
        <a class="Documentation-idLink" href="#{{.Name}}" aria-label="Go to {{.Name}}">¶</a>
      </h4>
      <div class="Documentation-declaration"><pre>{{.Decl}}</pre></div>
      {{para .Doc}}
      {{- with .Consts}}
      <div class="Documentation-typeConsts">
        <div class="Documentation-declaration"><pre>const {{$t.Name}} (
{{range .}}    {{.Decl}}{{with .Doc}} // {{.}}{{end}}
{{end}})</pre></div>
      </div>
      {{- end}}
      {{- range .Methods}}
      <div class="Documentation-typeMethod">
        <h4 tabindex="-1" id="{{$t.Name}}.{{.Name}}" data-kind="method" class="Documentation-typeMethodHeader">
          <span>{{$t.Name}}.<a class="Documentation-source" href="{{$.Src .File .Line}}">{{.Name}}</a></span>
          <a class="Documentation-idLink" href="#{{$t.Name}}.{{.Name}}" aria-label="Go to {{$t.Name}}.{{.Name}}">¶</a>
        </h4>
        <div class="Documentation-declaration"><pre>{{.Decl}}</pre></div>
        {{para .Doc}}
      </div>
      {{- end}}
    </div>
  {{- else}}
    <p class="Documentation-empty">No exported types.</p>
  {{- end}}
  </section>
{{- end}}
</div>`))

var lyubaOutline = template.Must(template.New("outline").Funcs(lyubaFuncs).Parse(`
<ul>
  {{- if .Doc}}<li><a href="#pkg-overview">Overview</a></li>{{end}}
  {{- if or .Consts .Types}}
  <li class="DocNav-overview"><a href="#pkg-index">Index</a></li>
  <li class="DocNav-constants"><a href="#pkg-constants">Constants</a></li>
  <li class="DocNav-types"><a href="#pkg-types">Types</a>
    <ul>
    {{- range .Types}}{{$t := .Name}}
      <li><a href="#{{.Name}}" title="type {{.Name}}">type {{.Name}}</a>
      {{- with .Methods}}<ul>{{range .}}<li><a href="#{{$t}}.{{.Name}}" title="{{first .Decl}}">{{first .Decl}}</a></li>{{end}}</ul>{{end}}
      </li>
    {{- end}}
    </ul>
  </li>
  {{- end}}
</ul>`))

var lyubaMobile = template.Must(template.New("mobile").Funcs(lyubaFuncs).Parse(`
<optgroup label="Documentation">
  {{- if .Doc}}<option value="pkg-overview">Overview</option>{{end}}
  {{- if or .Consts .Types}}<option value="pkg-index">Index</option>{{end}}
  {{- if .Consts}}<option value="pkg-constants">Constants</option>{{end}}
</optgroup>
{{- if .Types}}
<optgroup label="Types">
  {{- range .Types}}{{$t := .Name}}
  <option value="{{.Name}}">type {{.Name}}</option>
  {{- range .Methods}}<option value="{{$t}}.{{.Name}}">{{first .Decl}}</option>{{end}}
  {{- end}}
</optgroup>
{{- end}}`))

// renderLyuba returns the three html parts of a Lyuba package page.
func (p *Package) renderLyuba(ctx context.Context, innerPath string, sourceInfo *source.Info) (*dochtml.Parts, error) {
	d, err := ParseLyuba(p.lyuba)
	if err != nil {
		return nil, err
	}
	v := lyubaView{Package: d, src: func(file string, line int) string {
		if sourceInfo == nil {
			return ""
		}
		return sourceInfo.LineURL(strings.TrimPrefix(innerPath+"/"+file, "/"), line)
	}}
	exec := func(t *template.Template) (string, error) {
		var b bytes.Buffer
		if err := t.Execute(&b, v); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	body, err := exec(lyubaBody)
	if err != nil {
		return nil, err
	}
	outline, err := exec(lyubaOutline)
	if err != nil {
		return nil, err
	}
	mobile, err := exec(lyubaMobile)
	if err != nil {
		return nil, err
	}
	// html/template has already escaped everything that comes from the sources
	return &dochtml.Parts{
		Body:          uncheckedconversions.HTMLFromStringKnownToSatisfyTypeContract(body),
		Outline:       uncheckedconversions.HTMLFromStringKnownToSatisfyTypeContract(outline),
		MobileOutline: uncheckedconversions.HTMLFromStringKnownToSatisfyTypeContract(mobile),
	}, nil
}
