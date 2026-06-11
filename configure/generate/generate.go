package generate

import (
	"fmt"
)

type EventingMode int

const (
	EventingNone   EventingMode = iota
	EventingKafka
	EventingQStash
	EventingBoth
)

func (m EventingMode) String() string {
	switch m {
	case EventingNone:
		return "none"
	case EventingKafka:
		return "kafka"
	case EventingQStash:
		return "qstash"
	case EventingBoth:
		return "both"
	default:
		return "unknown"
	}
}

type Config struct {
	Eventing    EventingMode
	Notifiers   []string
	Deployments []string
}

var RootDir = "."

func All(cfg Config) error {
	fmt.Println()
	fmt.Println("Generating files...")

	if err := writeEnvFile(cfg); err != nil {
		return fmt.Errorf("env: %w", err)
	}
	fmt.Println("  ✔ .env.example")

	for _, dep := range cfg.Deployments {
		switch dep {
		case "local-compose", "prod-compose":
			if err := writeDockerCompose(cfg, dep); err != nil {
				return fmt.Errorf("docker-compose (%s): %w", dep, err)
			}
			fmt.Printf("  ✔ docker-compose.%s\n", composeFilename(dep))
		case "helm":
			if err := writeHelmValues(cfg); err != nil {
				return fmt.Errorf("helm: %w", err)
			}
			fmt.Println("  ✔ helm/manga-cdc/values-override.yaml")
		case "terraform":
			if err := writeTerraformTfvars(cfg); err != nil {
				return fmt.Errorf("terraform: %w", err)
			}
			fmt.Println("  ✔ terraform/terraform.tfvars")
		}
	}

	if err := writeGuide(cfg); err != nil {
		return fmt.Errorf("guide: %w", err)
	}
	fmt.Println("  ✔ SETUP.md")

	return nil
}

func composeFilename(dep string) string {
	if dep == "local-compose" {
		return "yml"
	}
	return "prod.yml"
}
