/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generator

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/gengo/args"
	"sigs.k8s.io/controller-tools/pkg/internal"
	"sigs.k8s.io/controller-tools/pkg/internal/parse"
	"sigs.k8s.io/controller-tools/pkg/util"
)

// Generator search API resource definition files,
// and generate CRD install files accordingly.
type Generator struct {
	RootPath          string
	Domain            string
	Namespace         string
	SkipMapValidation bool
}

func (c *Generator) ValidateAndInitFields() error {
	var err error

	if len(c.RootPath) == 0 {
		c.RootPath, err = filepath.Abs(".")
		if err != nil {
			return err
		}
	}

	// If Domain is not explicitly specified,
	// try to search for PROJECT file as a basis.
	if len(c.Domain) == 0 {
		for {
			if pathHasProjectFile(c.RootPath) {
				break
			}

			if isGoPath(c.RootPath) {
				return fmt.Errorf("failed to find working directory that contains %s", "PROJECT")
			}

			c.RootPath = path.Dir(c.RootPath)
		}
	}

	// Validate apis directory exists under working path
	apisPath := path.Join(c.RootPath, "pkg/apis")
	if _, err := os.Stat(apisPath); err != nil {
		fmt.Errorf("error validating apis path %s: %v", apisPath, err)
	}

	// Fetch domain from PROJECT file if not specified
	if len(c.Domain) == 0 {
		projectFile := path.Join(c.RootPath, "PROJECT")
		if _, err := os.Stat(projectFile); err != nil {
			fmt.Errorf("domain not specified and PROJECT file not found")
		}
		c.Domain = getDomainFromProject()
	}

	return nil
}

func (c *Generator) Do() error {
	arguments := args.Default()
	b, err := arguments.NewBuilder()
	if err != nil {
		return fmt.Errorf("failed making a parser: %v", err)
	}

	// Switch working directory to root path.
	if err := os.Chdir(c.RootPath); err != nil {
		return fmt.Errorf("failed switching working dir: %v", err)
	}

	if err := b.AddDirRecursive("./pkg/apis"); err != nil {
		return fmt.Errorf("failed making a parser: %v", err)
	}
	ctx, err := parse.NewContext(b)
	if err != nil {
		return fmt.Errorf("failed making a context: %v", err)
	}

	arguments.CustomArgs = &parse.ParseOptions{SkipMapValidation: c.SkipMapValidation}

	p := parse.NewAPIs(ctx, arguments, c.Domain)

	crds := c.getCrds(p)

	// Ensure output dir exists.
	outputPath := path.Join(c.RootPath, "config/crds")
	if err := os.MkdirAll(outputPath, os.FileMode(0700)); err != nil {
		return err
	}

	for file, crd := range crds {
		outFile := path.Join(outputPath, file)
		f, err := util.NewWriteCloser(outFile)
		if err != nil {
			return fmt.Errorf("failed to create %s: %v", outFile, err)
		}

		if c, ok := f.(io.Closer); ok {
			defer func() {
				if err := c.Close(); err != nil {
					log.Fatal(err)
				}
			}()
		}

		_, err = f.Write([]byte(crd))
		if err != nil {
			return fmt.Errorf("failed to write %s: %v", outFile, err)
		}
	}

	return nil
}

func pathHasProjectFile(filePath string) bool {
	if _, err := os.Stat(path.Join(filePath, "PROJECT")); os.IsNotExist(err) {
		return false
	}

	return true
}

func isGoPath(filePath string) bool {
	goPaths := strings.Split(os.Getenv("GOPATH"), ":")

	for _, path := range goPaths {
		if filePath == path {
			return true
		}
	}

	return false
}

func getCRDFileName(resource *codegen.APIResource) string {
	elems := []string{resource.Group, resource.Version, strings.ToLower(resource.Kind)}
	return strings.Join(elems, "_") + ".yaml"
}

func (c *Generator) getCrds(p *parse.APIs) map[string]string {
	crds := map[string]extensionsv1beta1.CustomResourceDefinition{}
	for _, g := range p.APIs.Groups {
		for _, v := range g.Versions {
			for _, r := range v.Resources {
				crd := r.CRD
				if len(c.Namespace) > 0 {
					crd.Namespace = c.Namespace
				}
				fileName := getCRDFileName(r)
				crds[fileName] = crd
			}
		}
	}

	result := map[string]string{}
	for file, crd := range crds {
		s, err := yaml.Marshal(crd)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		result[file] = string(s)
	}

	return result
}

func getDomainFromProject() string {
	var domain string

	file, err := os.Open("./PROJECT")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "domain:") {
			domainInfo := strings.Split(scanner.Text(), ":")
			if len(domainInfo) != 2 {
				log.Fatalf("Unexpected domain info: %s", scanner.Text())
			}
			domain = strings.Replace(domainInfo[1], " ", "", -1)
			break
		}
	}

	return domain
}
