package main

import (
	"fmt"
	"os"

	"github.com/aeswibon/manga-cdc/configure/generate"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║       manga-cdc Setup Wizard            ║")
	fmt.Println("║  Configure your eventing, notifiers,    ║")
	fmt.Println("║  and deployment in a few steps.         ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	cfg, err := runWizard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := generate.All(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! Run `cat SETUP.md` for next steps.")
}
