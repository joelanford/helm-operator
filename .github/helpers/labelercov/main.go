package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/godo.v2/glob"
	"sigs.k8s.io/yaml"
)

type File struct {
	Name   string
	Labels []string
}

func main() {
	var tree []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(path, ".git/") || info.IsDir() {
			return nil
		}

		tree = append(tree, path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	d, err := ioutil.ReadFile(".github/labeler.yml")
	if err != nil {
		log.Fatal(err)
	}

	var labeler map[string][]interface{}
	if err := yaml.Unmarshal(d, &labeler); err != nil {
		log.Fatal(err)
	}

	files := make(map[string]*File)
	for _, f := range tree {
		files[f] = &File{Name: f}
	}

	for label, spec := range labeler {
		for _, item := range spec {
			switch item := item.(type) {
			case string:
				filenames, err := filterTree(tree, item)
				if err != nil {
					log.Fatal(err)
				}

				for _, filename := range filenames {
					f := files[filename]
					f.Labels = append(f.Labels, label)
				}
			case map[string]interface{}:
				if any, ok := item["any"]; ok {
					any := toStringSlice(any.([]interface{}))
					filenames, err := filterTree(tree, any...)
					if err != nil {
						log.Fatal(err)
					}
					for _, filename := range filenames {
						f := files[filename]
						f.Labels = append(f.Labels, label)
					}

				} else if all, ok := item["all"]; ok {
					all := toStringSlice(all.([]interface{}))
					filenames, err := filterTree(tree, all...)
					if err != nil {
						log.Fatal(err)
					}
					for _, filename := range filenames {
						f := files[filename]
						f.Labels = append(f.Labels, label)
					}
				}
			}
		}
	}
	for _, file := range files {
		fmt.Printf("%s: %v\n", file.Name, file.Labels)
	}
}

func filterTree(tree []string, globStrings ...string) ([]string, error) {
	_, globs, err := glob.Glob(globStrings)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, filename := range tree {
		if matchGlobs(filename, globs) {
			out = append(out, filename)
		}
	}
	return out, nil
}

func matchGlobs(filename string, globs []*glob.RegexpInfo) bool {
	for _, gl := range globs {
		match := gl.MatchString(filename) != gl.Negate
		if !match {
			return false
		}
	}
	return true
}

func toStringSlice(slice []interface{}) (out []string) {
	for _, v := range slice {
		out = append(out, v.(string))
	}
	return out
}
