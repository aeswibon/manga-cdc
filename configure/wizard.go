package main

import (
	"fmt"
	"strings"

	"github.com/aeswibon/manga-cdc/configure/generate"
)

func runWizard() (generate.Config, error) {
	var cfg generate.Config

	fmt.Println("1/3  Eventing backend:")
	fmt.Println("     [1] None (DB polling only)")
	fmt.Println("     [2] Kafka")
	fmt.Println("     [3] QStash + Caddy (recommended)")
	fmt.Println("     [4] Both Kafka + QStash")
	choice, err := askOne("     Choose [1-4]", "3", func(s string) bool {
		return s >= "1" && s <= "4"
	})
	if err != nil {
		return cfg, err
	}
	switch choice {
	case "1":
		cfg.Eventing = generate.EventingNone
	case "2":
		cfg.Eventing = generate.EventingKafka
	case "3":
		cfg.Eventing = generate.EventingQStash
	case "4":
		cfg.Eventing = generate.EventingBoth
	}
	fmt.Println()

	fmt.Println("2/3  Notification channels (comma-separated, e.g. 1,2,3):")
	fmt.Println("     [1] Discord")
	fmt.Println("     [2] Slack")
	fmt.Println("     [3] Telegram")
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
		return cfg, err
	}
	notifierMap := map[string]string{"1": "discord", "2": "slack", "3": "telegram"}
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		if v, ok := notifierMap[r]; ok {
			cfg.Notifiers = append(cfg.Notifiers, v)
		}
	}
	fmt.Println()

	fmt.Println("3/3  Deployment targets (comma-separated, e.g. 1,2,3,4):")
	fmt.Println("     [1] Docker Compose (local dev)")
	fmt.Println("     [2] Docker Compose (production)")
	fmt.Println("     [3] Kubernetes / Helm")
	fmt.Println("     [4] Terraform + GCP")
	raw, err = askOne("     Choose", "1,2", func(s string) bool {
		for _, r := range strings.Split(s, ",") {
			r = strings.TrimSpace(r)
			if r < "1" || r > "4" {
				return false
			}
		}
		return true
	})
	if err != nil {
		return cfg, err
	}
	deployMap := map[string]string{"1": "local-compose", "2": "prod-compose", "3": "helm", "4": "terraform"}
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		if v, ok := deployMap[r]; ok {
			cfg.Deployments = append(cfg.Deployments, v)
		}
	}
	fmt.Println()

	return cfg, nil
}
