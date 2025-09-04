//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	r        = regexp.MustCompile(`^([^/]+)/([^/]+)/module\.go$`)
	tmplName = "builder.tmpl"
	outName  = "builder.go"
)

type Import struct {
	Vendor string
	Module string
}

type Data struct {
	Vendors map[string][]string
	Imports []Import
}

func TitleASCII(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	if rs[0] >= 'a' && rs[0] <= 'z' {
		rs[0] = rs[0] - ('a' - 'A')
	}
	return string(rs)
}

func main() {
	modules := make(map[string][]string)

	err := filepath.WalkDir("./", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && (path == "./generator" || path == "generator") {
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		slashPath := filepath.ToSlash(strings.TrimPrefix(path, "./"))

		if !r.MatchString(slashPath) {
			return nil
		}

		match := r.FindStringSubmatch(slashPath)
		vendor := "modules"
		mod := match[2]

		modules[vendor] = append(modules[vendor], mod)
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("walk error: %v", err))
	}

	for v, list := range modules {
		sort.Strings(list)
		modules[v] = list
	}

	imports := make([]Import, 0)
	for v, list := range modules {
		for _, m := range list {
			imports = append(imports, Import{Vendor: v, Module: m})
		}
	}

	// Keep import order stable
	sort.Slice(imports, func(i, j int) bool {
		if imports[i].Vendor == imports[j].Vendor {
			return imports[i].Module < imports[j].Module
		}
		return imports[i].Vendor < imports[j].Vendor
	})

	data := Data{Vendors: modules, Imports: imports}

	t, err := template.New(tmplName).
		Funcs(template.FuncMap{"Title": TitleASCII}).
		ParseFiles(filepath.Join("generator", tmplName))
	if err != nil {
		panic(fmt.Sprintf("failed to parse builder template: %s", err))
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("failed to generate %s content: %s", outName, err))
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		panic(fmt.Errorf("failed to format generated code: %w\n---\n%s", err, buf.String()))
	}

	if err := os.WriteFile(outName, formatted, 0o644); err != nil {
		panic(fmt.Sprintf("failed to write %s: %s", outName, err))
	}

	fmt.Printf("%s file successfully generated\n", outName)
}
