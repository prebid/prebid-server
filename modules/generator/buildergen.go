//go:build ignore

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"text/template"
)

var (
	r        = regexp.MustCompile("^([^/]+)/module.go$")
	tmplName = "builder.tmpl"
	outName  = "builder.go"
)

func main() {
	var modules []string

	filepath.WalkDir("./", func(path string, d fs.DirEntry, err error) error {
		if !r.MatchString(path) {
			return nil
		}
		modules = append(modules, r.FindStringSubmatch(path)[1])
		return nil
	})

	if len(modules) == 0 {
		fmt.Println("No modules found.")
		return
	}

	t, err := template.New(tmplName).ParseFiles(fmt.Sprintf("generator/%s", tmplName))
	if err != nil {
		panic(fmt.Sprintf("failed to parse builder template: %s", err))
	}

	f, err := os.Create(outName)
	if err != nil {
		panic(fmt.Sprintf("failed to create %s file: %s", outName, err))
	}
	defer f.Close()

	sort.Strings(modules)
	if err = t.Execute(f, modules); err != nil {
		panic(fmt.Sprintf("failed to generate %s file content: %s", outName, err))
	}

	fmt.Printf("%s file successfully generated\n", outName)
}
