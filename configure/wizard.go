package main

import (
	"fmt"
	"strings"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func runWizard() (manifest.Manifest, error) {
	fmt.Println("1/5  Deployment tier:")
	fmt.Println("     [1] Local development (Docker provisions Postgres + Redpanda)")
	fmt.Println("     [2] Production (bring your own managed services)")
	choice, err := askOne("     Choose [1-2]", "1", func(s string) bool {
		return s == "1" || s == "2"
	})
	if err != nil {
		return manifest.Manifest{}, err
	}
	fmt.Println()

	var m manifest.Manifest
	if choice == "1" {
		m = manifest.DefaultLocal()
	} else {
		m, err = runProductionWizard()
		if err != nil {
			return manifest.Manifest{}, err
		}
	}

	m.Notifiers, err = askNotifiers()
	if err != nil {
		return manifest.Manifest{}, err
	}

	if len(m.Notifiers) == 0 {
		fmt.Println("  Warning: no notification channels selected.")
	}
	fmt.Println()

	return m, nil
}

func runProductionWizard() (manifest.Manifest, error) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
		},
	}

	fmt.Println("2/5  Database provider preset:")
	fmt.Println("     [1] Generic (paste connection string)")
	fmt.Println("     [2] Aiven")
	fmt.Println("     [3] Neon")
	fmt.Println("     [4] Amazon RDS")
	choice, err := askOne("     Choose [1-4]", "1", func(s string) bool {
		return s >= "1" && s <= "4"
	})
	if err != nil {
		return m, err
	}
	presetMap := map[string]string{"1": "generic", "2": "aiven", "3": "neon", "4": "rds"}
	m.Database.Preset = presetMap[choice]

	url, err := askOne("     DATABASE_URL (postgres://...)", "", func(s string) bool {
		return strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://")
	})
	if err != nil {
		return m, err
	}
	m.Database.URL = url
	fmt.Println()

	fmt.Println("3/5  Eventing backend:")
	fmt.Println("     [1] Kafka (managed)")
	fmt.Println("     [2] QStash (HTTP webhook)")
	fmt.Println("     [3] None (advanced — DB only, no notifications via events)")
	choice, err = askOne("     Choose [1-3]", "1", func(s string) bool {
		return s >= "1" && s <= "3"
	})
	if err != nil {
		return m, err
	}
	switch choice {
	case "1":
		m.Eventing.Backend = manifest.EventingKafka
		if err := askKafkaConfig(&m); err != nil {
			return m, err
		}
	case "2":
		m.Eventing.Backend = manifest.EventingQStash
		if err := askQStashConfig(&m); err != nil {
			return m, err
		}
	case "3":
		m.Eventing.Backend = manifest.EventingNone
	}
	fmt.Println()

	fmt.Println("4/5  Deployment targets (comma-separated, e.g. 1,2,3):")
	fmt.Println("     [1] Docker Compose (production)")
	fmt.Println("     [2] Kubernetes / Helm")
	fmt.Println("     [3] Cloud infrastructure (Terraform)")
	raw, err := askOne("     Choose", "1", func(s string) bool {
		for _, r := range strings.Split(s, ",") {
			r = strings.TrimSpace(r)
			if r < "1" || r > "3" {
				return false
			}
		}
		return true
	})
	if err != nil {
		return m, err
	}
	deployMap := map[string]string{"1": "docker-compose-prod", "2": "helm", "3": "terraform"}
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		if v, ok := deployMap[r]; ok {
			m.Deploy.Targets = append(m.Deploy.Targets, v)
		}
	}
	fmt.Println()

	hasTerraform := false
	for _, t := range m.Deploy.Targets {
		if t == "terraform" {
			hasTerraform = true
			break
		}
	}
	if hasTerraform {
		fmt.Println("     Compute size preset:")
		fmt.Println("     [1] Micro  (1-2 vCPU, 1GB RAM - e.g. e2-micro, t3.micro, Standard_B1s, s-1vcpu-1gb) [Low Cost / Free Tier]")
		fmt.Println("     [2] Small  (1-2 vCPU, 2GB RAM - e.g. e2-small, t3.small, Standard_B1ms, s-1vcpu-2gb)")
		fmt.Println("     [3] Medium (2 vCPU, 4GB RAM   - e.g. e2-medium, t3.medium, Standard_B2s, s-2vcpu-2gb) [Recommended]")
		fmt.Println("     [4] Large  (4 vCPU, 8GB RAM   - e.g. e2-standard-2, t3.large, Standard_B2ms, s-4vcpu-8gb)")
		sizeChoice, err := askOne("     Choose [1-4]", "3", func(s string) bool {
			return s >= "1" && s <= "4"
		})
		if err != nil {
			return m, err
		}
		sizeMap := map[string]string{"1": "micro", "2": "small", "3": "medium", "4": "large"}
		m.Deploy.ComputeSize = sizeMap[sizeChoice]
		fmt.Println()
	}

	return m, nil
}

func askKafkaConfig(m *manifest.Manifest) error {
	fmt.Println("     Kafka preset:")
	fmt.Println("     [1] Generic")
	fmt.Println("     [2] Aiven")
	fmt.Println("     [3] Upstash Kafka")
	choice, err := askOne("     Choose [1-3]", "1", func(s string) bool {
		return s >= "1" && s <= "3"
	})
	if err != nil {
		return err
	}
	presetMap := map[string]string{"1": "generic", "2": "aiven", "3": "upstash-kafka"}
	m.Eventing.Preset = presetMap[choice]

	brokers, err := askOne("     KAFKA_BROKERS", "", func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	if err != nil {
		return err
	}
	m.Eventing.Kafka.Brokers = brokers
	m.Eventing.Kafka.Topic = "mangacdc.public.chapters"

	user, err := askOne("     KAFKA_USERNAME (optional)", "", func(string) bool { return true })
	if err != nil {
		return err
	}
	m.Eventing.Kafka.Username = user

	pass, err := askOne("     KAFKA_PASSWORD (optional)", "", func(string) bool { return true })
	if err != nil {
		return err
	}
	m.Eventing.Kafka.Password = pass

	if (user == "" || pass == "") && m.Eventing.Preset != "generic" {
		fmt.Println("  Warning: SASL presets usually require username and password.")
	}
	return nil
}

func askQStashConfig(m *manifest.Manifest) error {
	fmt.Println("     QStash preset:")
	fmt.Println("     [1] Generic")
	fmt.Println("     [2] Upstash QStash")
	choice, err := askOne("     Choose [1-2]", "1", func(s string) bool {
		return s == "1" || s == "2"
	})
	if err != nil {
		return err
	}
	presetMap := map[string]string{"1": "generic", "2": "upstash-qstash"}
	m.Eventing.Preset = presetMap[choice]

	token, err := askOne("     QSTASH_TOKEN", "", func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	if err != nil {
		return err
	}
	m.Eventing.QStash.Token = token

	dest, err := askOne("     QSTASH_DESTINATION_URL (https://...)", "", func(s string) bool {
		return strings.HasPrefix(s, "https://")
	})
	if err != nil {
		return err
	}
	m.Eventing.QStash.DestinationURL = dest
	return nil
}

func askNotifiers() ([]string, error) {
	fmt.Println("5/5  Notification channels (comma-separated, e.g. 1,2,3):")
	fmt.Println("     [1] Discord")
	fmt.Println("     [2] Slack")
	fmt.Println("     [3] Telegram")
	raw, err := askOne("     Choose", "1", func(s string) bool {
		if strings.TrimSpace(s) == "" {
			return true
		}
		for _, r := range strings.Split(s, ",") {
			r = strings.TrimSpace(r)
			if r < "1" || r > "3" {
				return false
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	notifierMap := map[string]string{"1": "discord", "2": "slack", "3": "telegram"}
	var out []string
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		if v, ok := notifierMap[r]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}
