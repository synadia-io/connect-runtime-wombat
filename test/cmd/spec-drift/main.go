package main

import (
	"fmt"
	"os"
)

func main() {
	run(".connect")
}

func run(dir string) {
	schemaDir, err := extract()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during extract: %v\n", err)
		os.Exit(1)
	}

	if err := validate(dir, schemaDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error during validate: %v\n", err)
	}

	if err := clean(schemaDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning schemas: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Removed extracted schemas from: %v\n", schemaDir)
	}

}

func clean(schemasDir string) error {
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(schemasDir); err != nil {
		return fmt.Errorf("failed to remove schemas directory: %w", err)
	}

	return nil
}
