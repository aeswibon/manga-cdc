package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func getComputeSizes(size string) (gcp, aws, azure, do string) {
	switch size {
	case "micro":
		return "e2-micro", "t3.micro", "Standard_B1s", "s-1vcpu-1gb"
	case "small":
		return "e2-small", "t3.small", "Standard_B1ms", "s-1vcpu-2gb"
	case "large":
		return "e2-standard-2", "t3.large", "Standard_B2ms", "s-4vcpu-8gb"
	case "medium":
		fallthrough
	default:
		return "e2-medium", "t3.medium", "Standard_B2s", "s-2vcpu-2gb"
	}
}

func renderCommonTfvars(m manifest.Manifest) string {
	var b strings.Builder

	b.WriteString("# Target Deployment Type: 'vm' (Docker Compose on VM), 'kubernetes' (Helm on Cluster), or 'serverless' (Serverless Container/Job)\n")
	b.WriteString("deployment_target = \"vm\"\n\n")

	b.WriteString("# Container Images\n")
	b.WriteString("scraper_image      = \"ghcr.io/aeswibon/manga-cdc/scraper:latest\"\n")
	b.WriteString("notification_image = \"ghcr.io/aeswibon/manga-cdc/notification-service:latest\"\n\n")

	b.WriteString("# Database Configuration\n")
	if m.Database.URL != "" {
		b.WriteString(fmt.Sprintf("database_url = %q\n\n", m.Database.URL))
	} else {
		b.WriteString("database_url = \"postgres://user:password@db-host:5432/mangacdc?sslmode=require\"\n\n")
	}

	b.WriteString("# Eventing / Streaming\n")
	if m.Eventing.Backend == manifest.EventingKafka {
		b.WriteString(fmt.Sprintf("kafka_brokers  = %q\n", m.Eventing.Kafka.Brokers))
		b.WriteString(fmt.Sprintf("kafka_username = %q\n", m.Eventing.Kafka.Username))
		b.WriteString(fmt.Sprintf("kafka_password = %q\n\n", m.Eventing.Kafka.Password))
	} else {
		b.WriteString("kafka_brokers  = \"\"\n")
		b.WriteString("kafka_username = \"\"\n")
		b.WriteString("kafka_password = \"\"\n\n")
	}

	b.WriteString("# Notification Webhooks and Tokens\n")
	discordVal := ""
	slackVal := ""
	tgToken := ""
	tgChat := ""

	for _, n := range m.Notifiers {
		switch n {
		case "discord":
			discordVal = "https://discord.com/api/webhooks/..."
		case "slack":
			slackVal = "https://hooks.slack.com/services/..."
		case "telegram":
			tgToken = "bot-token-here"
			tgChat = "chat-id-here"
		}
	}

	b.WriteString(fmt.Sprintf("discord_webhook_url = %q\n", discordVal))
	b.WriteString(fmt.Sprintf("slack_webhook_url   = %q\n", slackVal))
	b.WriteString(fmt.Sprintf("telegram_bot_token  = %q\n", tgToken))
	b.WriteString(fmt.Sprintf("telegram_chat_id    = %q\n\n", tgChat))

	b.WriteString("# Observability (Grafana Cloud / Alloy)\n")
	b.WriteString("observability_mode            = \"grafana-cloud\"\n")
	b.WriteString("grafana_cloud_prometheus_url  = \"\"\n")
	b.WriteString("grafana_cloud_prometheus_user = \"\"\n")
	b.WriteString("grafana_cloud_api_key         = \"\"\n")
	b.WriteString("grafana_cloud_stack_url       = \"\"\n")

	return b.String()
}

func writeTerraformConfigs(m manifest.Manifest) error {
	common := renderCommonTfvars(m)
	gcpSize, awsSize, azureSize, doSize := getComputeSizes(m.Deploy.ComputeSize)

	// GCP Setup
	var gcp strings.Builder
	gcp.WriteString("# GCP Provider Configuration\n")
	gcp.WriteString("project_id = \"your-gcp-project-id\"\n")
	gcp.WriteString("region     = \"us-central1\"\n")
	gcp.WriteString("zone       = \"us-central1-a\"\n\n")
	gcp.WriteString("# Compute Sizing\n")
	gcp.WriteString(fmt.Sprintf("machine_type  = %q\n", gcpSize))
	gcp.WriteString(fmt.Sprintf("gke_node_type = %q\n\n", gcpSize))
	gcp.WriteString(common)
	if err := writePlainFile(filepath.Join("terraform", "gcp", "terraform.tfvars.example"), gcp.String()); err != nil {
		return err
	}

	// AWS Setup
	var aws strings.Builder
	aws.WriteString("# AWS Provider Configuration\n")
	aws.WriteString("region = \"us-east-1\"\n\n")
	aws.WriteString("# Compute Sizing\n")
	aws.WriteString(fmt.Sprintf("instance_type = %q\n", awsSize))
	aws.WriteString(fmt.Sprintf("eks_node_type = %q\n\n", awsSize))
	aws.WriteString(common)
	if err := writePlainFile(filepath.Join("terraform", "aws", "terraform.tfvars.example"), aws.String()); err != nil {
		return err
	}

	// Azure Setup
	var azure strings.Builder
	azure.WriteString("# Azure Provider Configuration\n")
	azure.WriteString("location       = \"East US\"\n")
	azure.WriteString("ssh_public_key = \"\"\n\n")
	azure.WriteString("# Compute Sizing\n")
	azure.WriteString(fmt.Sprintf("vm_size       = %q\n", azureSize))
	azure.WriteString(fmt.Sprintf("aks_node_size = %q\n\n", azureSize))
	azure.WriteString(common)
	if err := writePlainFile(filepath.Join("terraform", "azure", "terraform.tfvars.example"), azure.String()); err != nil {
		return err
	}

	// DigitalOcean Setup
	var do strings.Builder
	do.WriteString("# DigitalOcean Provider Configuration\n")
	do.WriteString("do_token = \"your-digitalocean-api-token\"\n")
	do.WriteString("region   = \"sfo3\"\n\n")
	do.WriteString("# Compute Sizing\n")
	do.WriteString(fmt.Sprintf("droplet_size   = %q\n", doSize))
	do.WriteString(fmt.Sprintf("doks_node_size = %q\n\n", doSize))
	do.WriteString(common)
	if err := writePlainFile(filepath.Join("terraform", "digitalocean", "terraform.tfvars.example"), do.String()); err != nil {
		return err
	}

	return nil
}
