package main

import (
	"fmt"
	"os"
	"sync"
)

func main() {
	run(".connect")
}

func run(dir string) {
	filenames, err := generateSpecFilenames(dir)
	if err != nil {
		fmt.Printf("❌Error generating spec filenames: %v\n", err)
		os.Exit(1)
	}

	errDir, err := os.MkdirTemp("", "syntax-errors-")
	if err != nil {
		fmt.Printf("❌Error creating temp dir for errors: %v\n", err)
		os.Exit(1)
	}

	tester := NewTester(errDir)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, filename := range filenames {
		wg.Add(1)
		sem <- struct{}{} // acquire a slot
		go func(tester *Tester, fn string) {
			defer wg.Done()
			defer func() { <-sem }() // release the slot

			if err := tester.TestComponent(filename); err != nil {
				fmt.Printf("❌%v\n", err)
				return
			}

			//fmt.Printf("✅%s\n", fn)
		}(tester, filename)
	}
	wg.Wait()
}

func generateSpecFilenames(dir string) ([]string, error) {
	sourcesDir := dir + "/sources"
	sinksDir := dir + "/sinks"

	sourceSpecs, err := listSpecsInDir(sourcesDir)
	if err != nil {
		return nil, err
	}

	sinkSpecs, err := listSpecsInDir(sinksDir)
	if err != nil {
		return nil, err
	}

	fmt.Printf("🆗Found %d source specs and %d sink specs\n", len(sourceSpecs), len(sinkSpecs))

	return append(sourceSpecs, sinkSpecs...), nil
}

func listSpecsInDir(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var specs []string
	for _, file := range files {
		if !file.IsDir() && (len(file.Name()) > 5 && file.Name()[len(file.Name())-5:] == ".yaml" || len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".yml") {
			specs = append(specs, dir+"/"+file.Name())
		}
	}
	return specs, nil
}
