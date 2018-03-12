//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

var (
	options struct {
		OutputFile   string
		TemplatesDir string

		Namespace       string
		Image           string
		ImagePullPolicy string
		ImageSHA256     bool
		OperatorName    string
		RBAC            bool
	}
	templateNames = []string{
		"rbac.yaml",
		"deployment.yaml",
	}
)

func init() {
	pflag.StringVar(&options.OutputFile, "output", "manifests/arango-operator.yaml", "Path of the generated manifest file")
	pflag.StringVar(&options.TemplatesDir, "templates-dir", "manifests/templates", "Directory containing manifest templates")
	pflag.StringVar(&options.Namespace, "namespace", "default", "Namespace in which the operator will be deployed")
	pflag.StringVar(&options.Image, "image", "arangodb/arangodb-operator:latest", "Fully qualified image name of the ArangoDB operator")
	pflag.StringVar(&options.ImagePullPolicy, "image-pull-policy", "IfNotPresent", "Pull policy of the ArangoDB operator image")
	pflag.BoolVar(&options.ImageSHA256, "image-sha256", true, "Use SHA256 syntax for image")
	pflag.StringVar(&options.OperatorName, "operator-name", "arango-operator", "Name of the ArangoDB operator deployment")
	pflag.BoolVar(&options.RBAC, "rbac", true, "Use role based access control")

	pflag.Parse()
}

func main() {
	// Check options
	if options.OutputFile == "" {
		log.Fatal("--output not specified.")
	}
	if options.Namespace == "" {
		log.Fatal("--namespace not specified.")
	}
	if options.Image == "" {
		log.Fatal("--image not specified.")
	}

	// Fetch image sha256
	if options.ImageSHA256 {
		cmd := exec.Command(
			"docker",
			"inspect",
			"--format={{index .RepoDigests 0}}",
			options.Image,
		)
		result, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(string(result))
			log.Fatalf("Failed to fetch image SHA256: %v", err)
		}
		options.Image = strings.TrimSpace(string(result))
	}

	// Process templates
	templateOptions := struct {
		Namespace              string
		OperatorName           string
		OperatorImage          string
		ImagePullPolicy        string
		ClusterRoleName        string
		ClusterRoleBindingName string
		RBAC                   bool
	}{
		Namespace:              options.Namespace,
		OperatorName:           options.OperatorName,
		OperatorImage:          options.Image,
		ImagePullPolicy:        options.ImagePullPolicy,
		ClusterRoleName:        "arango-operator",
		ClusterRoleBindingName: "arango-operator",
		RBAC: options.RBAC,
	}
	output := &bytes.Buffer{}
	for i, name := range templateNames {
		t, err := template.New(name).ParseFiles(filepath.Join(options.TemplatesDir, name))
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}
		if i > 0 {
			output.WriteString("\n---\n\n")
		}
		output.WriteString(fmt.Sprintf("## %s\n", name))
		t.Execute(output, templateOptions)
		output.WriteString("\n")
	}

	// Save output
	outputPath, err := filepath.Abs(options.OutputFile)
	if err != nil {
		log.Fatalf("Failed to get absolute output path: %v\n", err)
	}
	if err := os.MkdirAll(filepath.Base(outputPath), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v\n", err)
	}
	if err := ioutil.WriteFile(outputPath, output.Bytes(), 0644); err != nil {
		log.Fatalf("Failed to write output file: %v\n", err)
	}
}
