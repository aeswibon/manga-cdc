package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aeswibon/manga-cdc/configure/generate"
	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func main() {
	configPath := flag.String("f", "manga-cdc.config.yaml", "manifest file path")
	generateFlag := flag.Bool("generate", false, "generate artifacts from an existing manifest")
	validateFlag := flag.Bool("validate", false, "validate manifest only")
	flag.Parse()

	cmd := ""
	if len(flag.Args()) > 0 {
		cmd = flag.Args()[0]
	}
	if *generateFlag {
		cmd = "generate"
	}
	if *validateFlag {
		cmd = "validate"
	}

	switch cmd {
	case "generate":
		if err := runGenerate(*configPath); err != nil {
			fail(err)
		}
		return
	case "validate":
		if err := runValidate(*configPath); err != nil {
			fail(err)
		}
		fmt.Println("Manifest is valid.")
		return
	case "help", "-h", "--help":
		printUsage()
		return
	}

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║       manga-cdc Setup Wizard            ║")
	fmt.Println("║  Configure tier, services, notifiers,   ║")
	fmt.Println("║  and deployment in a few steps.         ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	m, err := runWizard()
	if err != nil {
		fail(err)
	}

	path := filepath.Join(generate.RootDir, *configPath)
	if err := manifest.Save(path, m); err != nil {
		fail(err)
	}
	fmt.Printf("  ✔ %s\n", *configPath)

	if err := generate.All(m); err != nil {
		fail(fmt.Errorf("generating files: %w", err))
	}

	fmt.Println()
	fmt.Println("Done! Run `cat SETUP.md` for next steps.")
}

func printUsage() {
	fmt.Println(`Usage:
  go run ./configure                 Interactive wizard
  go run ./configure generate        Generate from manga-cdc.config.yaml
  go run ./configure generate -f X   Generate from alternate manifest
  go run ./configure validate        Validate manifest only`)
}

func runGenerate(configPath string) error {
	path := filepath.Join(generate.RootDir, configPath)
	return generate.AllFromFile(path)
}

func runValidate(configPath string) error {
	path := filepath.Join(generate.RootDir, configPath)
	m, err := manifest.Load(path)
	if err != nil {
		return err
	}
	return m.Validate()
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
