package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/synadia-io/connect-runtime-wombat/tools/docs_gen/corrections"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"

	"github.com/redpanda-data/benthos/v4/public/service"

	_ "github.com/synadia-io/connect-runtime-wombat/components"

	_ "embed"
)

var IncludedInputs = []string{
	"amqp_0_9", "amqp_1",
}

var IncludedOutputs = []string{
	"amqp_0_9", "amqp_1",
}

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
	env.WalkInputs(viewForDir(config, path.Join(docsDir, "./components/inputs")))
	env.WalkScanners(viewForDir(config, path.Join(docsDir, "./components/scanners")))
	env.WalkOutputs(viewForDir(config, path.Join(docsDir, "./components/outputs")))
}

func viewForDir(cfg *Config, docsDir string) func(string, *service.ConfigView) {
	return func(name string, view *service.ConfigView) {

		data, err := view.TemplateData()
		if err != nil {
			panic(fmt.Sprintf("Failed to prepare docs for '%v': %v", name, err))
		}

		if !cfg.Contains(data.Type, data.Name) {
			fmt.Printf("Skipping docs for '%v' as it is not in the config\n", name)
			return
		}

		result, err := Generate(data)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate docs for '%v': %v", name, err))
		}
		if err := os.MkdirAll(docsDir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create docs directory path '%v': %v", docsDir, err))
		}

		b, err := yaml.Marshal(result)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal docs for '%v': %v", name, err))
		}

		var root map[string]any
		_ = yaml.Unmarshal(b, &root)

		doc := gabs.Wrap(root)

		a2m := corrections.AsciidocToMd{Path: "description"}
		doc, err = a2m.Correct(doc)
		if err != nil {
			panic(fmt.Sprintf("Failed to correct docs for '%v': %v", name, err))
		}

		b, err = yaml.Marshal(doc.Data())
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal corrected docs for '%v': %v", name, err))
		}

		create(name, path.Join(docsDir, name+".yaml"), b)
	}
}
