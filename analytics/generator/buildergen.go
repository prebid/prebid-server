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
	Module string
}

type Data struct {
	Modules []string
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
	modSet := make(map[string]struct{})

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

	data := Data{Modules: modules}

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
