package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/Jeffail/gabs/v2"
	"github.com/synadia-io/connect-runtime-wombat/tools/docs_gen/corrections"
	"gopkg.in/yaml.v3"

	"github.com/redpanda-data/benthos/v4/public/service"

	_ "github.com/synadia-io/connect-runtime-wombat/components"

	_ "embed"
)

var Version = "0.0.1"
var DateBuilt = "1970-01-01T00:00:00Z"

func create(t, path string, resBytes []byte) {
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, resBytes) {
			return
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}

	if err := os.WriteFile(path, resBytes, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Documentation for '%v' has changed, updating: %v\n", t, path)
}

func getSchema() *service.ConfigSchema {
	env := service.GlobalEnvironment()
	s := env.FullConfigSchema(Version, DateBuilt)

	s.SetFieldDefault(map[string]any{
		"@service": "wombat",
	}, "logger", "static_fields")

	return s
}

func main() {
	docsDir := "./target/docs"
	flag.StringVar(&docsDir, "dir", docsDir, "The directory to write docs to")
	flag.Parse()

	// -- get the config dir
	if flag.NArg() < 1 {
		panic("Must provide a config dir as the first argument")
	}

	configDir := flag.Args()[0]
	if configDir == "" {
		panic("Must provide a config directory")
	}

	// -- load the config
	config := &Config{}
	if err := LoadConfig(config, configDir); err != nil {
		panic(err)
	}

	env := getSchema().Environment()
	env.WalkInputs(viewForDir(config, path.Join(docsDir, "./sources"), "input"))
	env.WalkScanners(viewForDir(config, path.Join(docsDir, "./scanners"), "scanner"))
	env.WalkOutputs(viewForDir(config, path.Join(docsDir, "./sinks"), "output"))
}

func viewForDir(cfg *Config, docsDir string, componentType string) func(string, *service.ConfigView) {
	return func(name string, view *service.ConfigView) {

		data, err := view.TemplateData()
		if err != nil {
			panic(fmt.Sprintf("Failed to prepare docs for '%v': %v", name, err))
		}

		if !cfg.Contains(data.Type, data.Name) {
			fmt.Printf("Skipping docs for '%v' as it is not in the config\n", name)
			return
		}

		result, err := Generate(data, componentType)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate docs for '%v': %v", name, err))
		}

		// Set icon from config if available
		if icon := cfg.GetIcon(componentType, name); icon != "" {
			result.Icon = &icon
		}
		if err := os.MkdirAll(docsDir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create docs directory path '%v': %v", docsDir, err))
		}

		b, err := yaml.Marshal(result)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal docs for '%v': %v", name, err))
		}

		// For SQL components, the YAML may have complex multiline strings that cause unmarshal issues
		// Let's try to work around this
		var root map[string]any
		if err := yaml.Unmarshal(b, &root); err != nil {
			// If unmarshal fails for SQL components, skip the AsciidocToMd processing
			if name == "sql_raw" || name == "sql_insert" || name == "sql_select" {
				fmt.Printf("WARNING: Failed to unmarshal %s, skipping AsciidocToMd: %v\n", name, err)
				// Just prepend model_version and save the original YAML
				modelVersionYAML := []byte("model_version: \"1\"\n")
				b = append(modelVersionYAML, b...)
				create(name, path.Join(docsDir, name+".yml"), b)
				return
			}
			panic(fmt.Sprintf("Failed to unmarshal docs for '%v': %v", name, err))
		}

		doc := gabs.Wrap(root)

		a2m := corrections.AsciidocToMd{Path: "description"}
		correctedDoc, err := a2m.Correct(doc)
		if err != nil {
			panic(fmt.Sprintf("Failed to correct docs for '%v': %v", name, err))
		}

		// If correction returned nil, keep the original doc
		if correctedDoc != nil {
			doc = correctedDoc
		}

		// Marshal the corrected doc first
		b, err = yaml.Marshal(doc.Data())
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal corrected docs for '%v': %v", name, err))
		}

		// Skip if the result is just "null"
		if string(b) == "null\n" {
			fmt.Printf("Skipping '%v' as it generated null content\n", name)
			return
		}

		// Prepend model_version: "1" to the YAML
		modelVersionYAML := []byte("model_version: \"1\"\n")
		b = append(modelVersionYAML, b...)

		create(name, path.Join(docsDir, name+".yml"), b)
	}
}
