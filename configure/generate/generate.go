package generate

import (
	"fmt"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func All(m manifest.Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Generating files...")

	if err := writeEnvFile(m); err != nil {
		return fmt.Errorf("env: %w", err)
	}
	fmt.Println("  ✔ .env.example")

	switch m.Tier {
	case manifest.TierLocal:
		if err := writeComposeLocal(m); err != nil {
			return fmt.Errorf("docker-compose: %w", err)
		}
		fmt.Println("  ✔ docker-compose.yml")
	case manifest.TierProduction:
		if m.HasDeployTarget("docker-compose-prod") {
			if err := writeComposeProd(m); err != nil {
				return fmt.Errorf("docker-compose prod: %w", err)
			}
			fmt.Println("  ✔ docker-compose.prod.yml")
			if m.Eventing.Backend == manifest.EventingQStash {
				fmt.Println("  ✔ Caddyfile")
			}
		}
		if m.HasDeployTarget("helm") {
			if err := writeHelmValues(m); err != nil {
				return fmt.Errorf("helm: %w", err)
			}
			fmt.Println("  ✔ helm/manga-cdc/values-override.yaml")
		}
		if m.HasDeployTarget("terraform") {
			if err := writeTerraformConfigs(m); err != nil {
				return fmt.Errorf("terraform: %w", err)
			}
			fmt.Println("  ✔ terraform/<cloud>/terraform.tfvars.example")
		}
	}

	if err := writeGuide(m); err != nil {
		return fmt.Errorf("guide: %w", err)
	}
	fmt.Println("  ✔ SETUP.md")

	return nil
}

func AllFromFile(path string) error {
	m, err := manifest.Load(path)
	if err != nil {
		return err
	}
	return All(m)
}
