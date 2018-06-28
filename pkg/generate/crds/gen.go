// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crds

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/gengo/args"
	"sigs.k8s.io/controller-tools/pkg/generate"
	"sigs.k8s.io/controller-tools/pkg/internal/codegen"
	"sigs.k8s.io/controller-tools/pkg/internal/codegen/parse"
)

type CRDGenerator struct {
	RootPath string
	Namespace string
	SkipMapValidation bool
}

func (c *CRDGenerator)GenerateCRDs() error {
	arguments := args.Default()
	b, err := arguments.NewBuilder()
	if err != nil {
		return fmt.Errorf("Failed making a parser: %v", err)
	}

	if len(c.RootPath) > 0 && !pathHasProjectFile(c.RootPath) {
		return fmt.Errorf("Input path must be a diretory containing %s", "PROJECT")
	}

	// Search for PROJECT if no rootPath provided
	if len(c.RootPath) == 0 {
		c.RootPath, err = filepath.Abs(".")
		if err != nil {
			return err
		}

		for {
			if pathHasProjectFile(c.RootPath) {
				break
			}

			if c.RootPath == "/" {
				return fmt.Errorf("failed to find working directory that contains %s", "PROJECT")
			}

			c.RootPath = path.Dir(c.RootPath)
		}
	}

	resourcePath := path.Join(c.RootPath, "pkg/apis")

	if err := b.AddDirRecursive(resourcePath); err != nil {
		return fmt.Errorf("Failed making a parser: %v", err)
	}
	ctx, err := parse.NewContext(b)
	if err != nil {
		return fmt.Errorf("Failed making a context: %v", err)
	}

	arguments.CustomArgs = &parse.ParseOptions{SkipMapValidation: c.SkipMapValidation}

	p := parse.NewAPIs(ctx, arguments)

	crds := c.getCrds(p)

	// Ensure output dir exists.
	outputPath := path.Join(c.RootPath, "config/crds")
	if err := os.MkdirAll(outputPath, os.FileMode(0700)); err != nil {
		return err
	}

	for file, crd := range crds {
		generate.WriteString(path.Join(outputPath, file), crd)
	}

	return nil
}

func pathHasProjectFile(filePath string) bool {
	if _, err := os.Stat(path.Join(filePath, "PROJECT")); os.IsNotExist(err) {
		return false
	}

	return true
}

func getCRDFileName(resource *codegen.APIResource) string {
	elems := []string{resource.Group, resource.Version, strings.ToLower(resource.Kind)}
	return strings.Join(elems, "_") + ".yaml"
}

func (c *CRDGenerator)getCrds(p *parse.APIs) map[string]string {
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
			glog.Fatalf("Error: %v", err)
		}
		result[file] = string(s)
	}

	return result
}