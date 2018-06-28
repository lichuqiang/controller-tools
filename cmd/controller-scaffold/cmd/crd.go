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

package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-tools/pkg/generate/crds"
)

var g *crds.CRDGenerator

// CRDCmd represents the crd command
var CRDCmd = &cobra.Command{
	Use:   "crd",
	Short: "Scaffold CRD for the resource definition",
	Long: `Scaffold CRD files for existing Kubernetes API resources under path pkg/apis.

Root path of the can be set via root-path flag. The path must have PROJECT file and resource
dir pkg/apis under it. Command will search parent directories for a working path that contains
the PROJECT file.

Output crd files can be found under path 'config/crds'' of the root path.
`,
	Example: "controller-scaffold crd --root-path /workspace/src/example.com",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Writing CRD files...")
		err := g.GenerateCRDs()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("CRD files generated, files can be found under path %s.\n", path.Join(g.RootPath, "config/crds"))
	},
}

func init() {
	rootCmd.AddCommand(CRDCmd)
	g = GeneratorForFlags(CRDCmd.Flags())
}

// GeneratorForFlags registers flags for CRDGenerator fields and returns the Generator.
func GeneratorForFlags(f *flag.FlagSet) *crds.CRDGenerator {
	g := &crds.CRDGenerator{}
	f.StringVar(&g.RootPath, "root-path", "", "working dir, must have PROJECT file under the path")
	// TODO: Do we need this? Is there a possiblility that a crd is namespace scoped?
	f.StringVar(&g.Namespace, "namespace", "", "CRD namespace, treat it as root scoped in not set")
	f.BoolVar(&g.SkipMapValidation, "skip-map-validation", true, "if set to true, skip generating validation schema for map type in CRD.")
	return g
}
