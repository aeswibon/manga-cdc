# Multi-Cloud Production Deployment Guide (Terraform IaC)

This directory contains production-ready, modular Terraform configurations to provision secure infrastructure and application runtimes across four major cloud providers:

1. **[GCP (Google Cloud Platform)](file:///Volumes/Seagate/developer/personal/manga-cdc/terraform/gcp)**
2. **[AWS (Amazon Web Services)](file:///Volumes/Seagate/developer/personal/manga-cdc/terraform/aws)**
3. **[Azure](file:///Volumes/Seagate/developer/personal/manga-cdc/terraform/azure)**
4. **[DigitalOcean](file:///Volumes/Seagate/developer/personal/manga-cdc/terraform/digitalocean)**

---

## 1. Supported Deployment Targets

Each cloud provider configuration supports three target modes via the `deployment_target` variable:

| Target Name | GCP | AWS | Azure | DigitalOcean |
| :--- | :--- | :--- | :--- | :--- |
| **`vm`** (Docker Compose) | Google Compute Engine | Amazon EC2 Instance | Azure Linux VM | DO Droplet |
| **`kubernetes`** (Helm) | Google Kubernetes Engine (GKE) | Elastic Kubernetes Service (EKS) | Azure Kubernetes Service (AKS) | DO Kubernetes (DOKS) |
| **`serverless`** (Managed) | Cloud Run (Service + Job) & Scheduler | ECS Fargate & EventBridge | Container Apps & Jobs | App Platform (Service + Worker) |

---

## 2. Target Specifics & Architecture

### VM Mode (`deployment_target = "vm"`)
* **Execution**: Provisions a single VM running Ubuntu 22.04 LTS. A unified startup script ([startup.sh.tftpl](file:///Volumes/Seagate/developer/personal/manga-cdc/terraform/templates/startup.sh.tftpl)) automatically installs Docker, registers system services, writes a local `.env` configuration, renders production Docker Compose configurations, and launches the scraper and notification services.
* **Storage**: Sets up an OS disk (Standard LRS, gp3, Standard_LRS, or Droplet block storage) default-sized at 30GB.
* **Garbage Collection**: Configures a system cron job to run `docker image prune -a -f --filter "until=24h"` daily to prevent disk bloat.
* **Security & Network Isolation**:
  * The scraper (`2112`), notification service (`8080`), self-hosted Prometheus (`9090`), and Grafana (`3000`) ports are bound strictly to `127.0.0.1` locally.
  * Public cloud firewalls / security groups strictly limit ingress to ports `22` (SSH), `80` (HTTP), and `443` (HTTPS) for TLS traffic proxying.

### Kubernetes Mode (`deployment_target = "kubernetes"`)
* **Execution**: Provisions a managed Kubernetes cluster with a worker node pool. Once active, Terraform utilizes the Kubernetes and Helm providers to deploy the application directly via the [helm/manga-cdc](file:///Volumes/Seagate/developer/personal/manga-cdc/helm/manga-cdc) package.
* **Scaling**: Configures rolling-updates to prevent downtime. The default configuration sets up 2 worker nodes.

### Serverless Mode (`deployment_target = "serverless"`)
Eliminates public IP address charges and minimizes compute costs to $0/month (or pay-per-execution) by deploying code as managed containers:
* **GCP**: Provisions a Google Cloud Run Service (notifiers) and a Cloud Run Job (scraper). A Google Cloud Scheduler triggers the scraper run-once job using a cron expression.
* **AWS**: Provisions an Amazon ECS Fargate Cluster. Notifiers run as a continuous Fargate service; the scraper runs as a one-shot Fargate task triggered periodically by an Amazon EventBridge Scheduler rule.
* **Azure**: Provisions an Azure Container App (notifier) and an Azure Container App Job (scraper) using a native scheduled trigger configuration.
* **DigitalOcean**: Provisions a DigitalOcean App Platform application containing a `service` component (notifier web server) and a `worker` daemon component (scraper running its internal sleep-scrape loop continuously).

---

## 3. Compute Sizes & Mapping Presets

Your compute size selection is mapped to cloud-native instance sizes:

| Preset | GCP (GCE/GKE) | AWS (EC2/EKS) | Azure (VM/AKS) | DigitalOcean |
| :--- | :--- | :--- | :--- | :--- |
| **`micro`** | `e2-micro` (Free-Tier) | `t3.micro` | `Standard_B1s` | `s-1vcpu-1gb` |
| **`small`** | `e2-small` | `t3.small` | `Standard_B1ms` | `s-1vcpu-2gb` |
| **`medium`** | `e2-medium` | `t3.medium` | `Standard_B2s` | `s-2vcpu-2gb` |
| **`large`** | `e2-standard-2` | `t3.large` | `Standard_B2ms` | `s-4vcpu-8gb` |

---

## 4. Security & Secret Management

* **Sensitive Variables**: Secrets like `database_url`, `kafka_password`, `grafana_cloud_api_key`, and webhook tokens are marked `sensitive = true` in Terraform to prevent leakage in shell outputs and plans.
* **Azure ACA Secrets**: On Azure Container Apps, sensitive values are declared in a `secret` block with Azure-compliant names (`lower(replace(key, "_", "-"))`). They are stored securely in the Azure portal and injected via `secret_name` references in the container environment rather than as plain-text configurations.
* **DigitalOcean App Platform Secrets**: Sensitive environment variables are registered with `type = "SECRET"`, indicating to DigitalOcean App Platform that they should be encrypted at rest and masked in app specifications.
* **IAM Roles**: Minimum privilege IAM roles are provisioned (e.g. AWS Task Execution Roles, GCP Service Accounts) to limit access boundaries.

---

## 5. Provider Authentications

Prior to running Terraform, ensure you have credentials configured locally:

### Google Cloud (GCP)
Run the following commands to authenticate:
```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project <your-project-id>
```

### Amazon Web Services (AWS)
Run `aws configure` and set your Access Key, Secret Key, and default region:
```bash
aws configure
```

### Microsoft Azure
Authenticate via the Azure CLI:
```bash
az login
```

### DigitalOcean
Generate an API token from your DigitalOcean Control Panel under **API -> Personal Access Tokens** and pass it via the `do_token` variable (or set the `TF_VAR_do_token` environment variable).

---

## 6. Local Command Line Workflow

1. Navigate to your cloud provider directory:
   ```bash
   cd terraform/azure # Or gcp, aws, digitalocean
   ```
2. Copy the example variables file:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```
3. Edit `terraform.tfvars` and configure your credentials, database connection URL, and image coordinates.
4. Execute the Terraform sequence:
   ```bash
   terraform init       # Initialize plugins
   terraform validate   # Validate HCL syntax
   terraform plan       # Preview resources to create
   terraform apply      # Execute deployment
   ```
5. View output URLs/IP addresses:
   * VM IP: `terraform output -raw vm_public_ip`
   * Serverless URL: `terraform output -raw container_app_url` (Azure) or `terraform output -raw app_platform_url` (DigitalOcean)

---

## 7. Unified GitHub Actions Deployment Pipeline

The repository includes a unified CI/CD workflow at `.github/workflows/deploy.yml` which deploys your application automatically on release tags (`v*`) or manually via the Actions UI.

### Deployment Methods (`deploy_method`)

The workflow supports two deployment methods, configured via the `DEPLOY_METHOD` secret/variable (defaults to `direct`) or selected manually via `workflow_dispatch`:

1. **`direct` (Default / Low-Friction)**:
   * Deploys container updates directly to your environment using SSH/SCP (for VMs), Helm (for Kubernetes), or cloud CLIs (for Serverless).
   * **No state setup required**: Ideal for quick setups as it does not touch your Terraform state.
2. **`terraform` (IaC / GitOps)**:
   * Executes `terraform init` and `terraform apply -auto-approve` inside the runner, applying both infrastructure and container updates together.
   * **Remote Backend Requirement**: To use this method, you **must** configure a remote backend (e.g. AWS S3, GCP GCS, Azure Blob, or Terraform Cloud) in your Terraform files to persist the state. Using the default local backend will cause state loss between runner executions, resulting in deployment failures.

### GitHub Repository Secrets
To enable automated deployments, configure the following secrets in your GitHub repository:

#### Terraform State Secrets (Optional / For `deploy_method: terraform`)
If you use the `terraform` deploy method, the pipeline can dynamically provision and manage your remote state storage bucket. Set up these secrets to let the pipeline handle the configuration automatically:
* `TF_STATE_BUCKET`: The name of the storage bucket (GCS for GCP, S3 for AWS, Space for DigitalOcean).
* `DO_SPACE_ENDPOINT`: The Space API endpoint (e.g. `sfo3.digitaloceanspaces.com`, required for DigitalOcean).
* `AZURE_STORAGE_ACCOUNT`: The Azure Storage Account name (required for Azure).

#### Core Secrets (All Targets)
* `DATABASE_URL`: PostgreSQL connection string (starts with `postgres://` or `postgresql://`).
* `KAFKA_BROKERS`, `KAFKA_USERNAME`, `KAFKA_PASSWORD`: Message streaming credentials (optional).
* `DISCORD_WEBHOOK_URL`, `SLACK_WEBHOOK_URL`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`: Notification variables (optional).

#### GCP Target Secrets
* `GCP_WORKLOAD_IDENTITY_PROVIDER`: Workload Identity Provider string (e.g., `projects/12345/locations/global/workloadIdentityPools/my-pool/providers/my-provider`).
* `GCP_SERVICE_ACCOUNT`: Service Account Email.
* `GCP_ZONE` / `GCP_REGION`: Target GCP Zone/Region.
* `GCP_SSH_USER` / `GCP_SSH_PRIVATE_KEY` / `GCP_VM_NAME`: Credentials for VM mode.
* `GKE_CLUSTER_NAME`: Target cluster for Kubernetes mode.

#### AWS Target Secrets
* `AWS_ACCESS_KEY_ID`: AWS Access Key.
* `AWS_SECRET_ACCESS_KEY`: AWS Secret Access Key.
* `AWS_REGION`: Target region.
* `SSH_PRIVATE_KEY` / `VM_USER` / `VM_HOST`: Credentials for VM mode.
* `EKS_CLUSTER_NAME`: Target cluster for Kubernetes mode.

#### Azure Target Secrets
* `AZURE_CREDENTIALS`: JSON output of `az ad sp create-for-rbac --sdk-auth` containing credentials.
* `SSH_PRIVATE_KEY` / `VM_USER` / `VM_HOST`: Credentials for VM mode.
* `AKS_CLUSTER_NAME` / `AZURE_RESOURCE_GROUP`: Credentials for Kubernetes mode.

#### DigitalOcean Target Secrets
* `DIGITALOCEAN_ACCESS_TOKEN`: DO API token.
* `SSH_PRIVATE_KEY` / `VM_USER` / `VM_HOST`: Credentials for Droplet VM mode.
* `DOKS_CLUSTER_NAME`: Cluster name for Kubernetes mode.
* `DO_APP_ID`: Application ID for Serverless mode (DO App Platform).
