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
	r        = regexp.MustCompile("^([^/]+)/([^/]+)/module.go$")
	tmplName = "builder.tmpl"
	outName  = "builder.go"
)

func main() {
	modules := make(map[string][]string)

	filepath.WalkDir("./", func(path string, d fs.DirEntry, err error) error {
		if !r.MatchString(path) {
			return nil
		}
		match := r.FindStringSubmatch(path)
		vendorModules := modules[match[1]]
		vendorModules = append(vendorModules, match[2])
		modules[match[1]] = vendorModules

		return nil
	})

	for vendorName, names := range modules {
		sort.Strings(names)
		modules[vendorName] = names
	}

	funcMap := template.FuncMap{"Title": strings.Title}
	t, err := template.New(tmplName).Funcs(funcMap).ParseFiles(fmt.Sprintf("generator/%s", tmplName))
	if err != nil {
		panic(fmt.Sprintf("failed to parse builder template: %s", err))
	}

	f, err := os.Create(outName)
	if err != nil {
		panic(fmt.Sprintf("failed to create %s file: %s", outName, err))
	}
	defer f.Close()

	var buf bytes.Buffer
	if err = t.Execute(&buf, modules); err != nil {
		panic(fmt.Sprintf("failed to generate %s file content: %s", outName, err))
	}

	content, err := format.Source(buf.Bytes())
	if err != nil {
		panic(fmt.Sprintf("failed to format generated code: %s", err))
	}

	if _, err = f.Write(content); err != nil {
		panic(fmt.Sprintf("failed to write file content: %s", err))
	}

	fmt.Printf("%s file successfully generated\n", outName)
}
