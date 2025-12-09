//go:build ignore

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
)

var (
	r        = regexp.MustCompile(`^([^/]+)/([^/]+)/module\.go$`)
	tmplName = "builder.tmpl"
	outName  = "builder.go"
)

//go:embed builder.tmpl
var builderTmpl string

type Import struct {
	Module string
}

type Data struct {
	Modules []string
	Imports []string
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

func analyticsRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(thisFile))
}

func main() {
	root := analyticsRoot()
	modSet := make(map[string]struct{})

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")
		if rel == "generator" || strings.HasPrefix(rel, "generator/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		if !r.MatchString(rel) {
			return nil
		}
		match := r.FindStringSubmatch(rel)
		mod := match[2]
		modSet[mod] = struct{}{}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("walk error: %v", err))
	}

	modules := make([]string, 0, len(modSet))
	for m := range modSet {
		modules = append(modules, m)
	}
	sort.Strings(modules)

	data := Data{
		Modules: modules,
		Imports: modules,
	}

	t, err := template.New(tmplName).
		Funcs(template.FuncMap{"Title": TitleASCII}).
		Parse(builderTmpl)
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

	outPath := filepath.Join(root, "build", outName)
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		panic(fmt.Sprintf("failed to write %s: %s", outPath, err))
	}

	fmt.Printf("%s file successfully generated at %s\n", outName, outPath)
}
