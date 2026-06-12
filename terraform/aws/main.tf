terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.region

  access_key                  = var.ci_plan_mode ? "ci-dummy-access-key" : null
  secret_key                  = var.ci_plan_mode ? "ci-dummy-secret-key" : null
  skip_credentials_validation = var.ci_plan_mode
  skip_requesting_account_id  = var.ci_plan_mode
}

data "aws_vpc" "default" {
  count   = var.ci_plan_mode ? 0 : 1
  default = true
}

data "aws_subnets" "default" {
  count = var.ci_plan_mode ? 0 : 1
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default[0].id]
  }
}

# -----------------------------------------------------------------------------
# Database URL Parser
# -----------------------------------------------------------------------------
locals {
  # Regex to parse connection string format: postgres://user:pass@host:port/dbname?query
  db_parts          = regex("^postgres://(?:(?P<user>[^:@]+)(?::(?P<pass>[^@]+))?@)?(?P<host>[^/]+)(?P<path>/[^?]+)?(?:\\?(?P<query>.+))?$", var.database_url)
  db_user           = local.db_parts.user != null ? local.db_parts.user : ""
  db_pass           = local.db_parts.pass != null ? local.db_parts.pass : ""
  db_host           = local.db_parts.host
  db_path           = local.db_parts.path != null ? local.db_parts.path : "/postgres"
  db_query          = local.db_parts.query != null ? "?${local.db_parts.query}" : ""
  db_path_and_query = "${local.db_path}${local.db_query}"

  aws_vpc_id     = var.ci_plan_mode ? "vpc-ci000000000000000" : data.aws_vpc.default[0].id
  aws_subnet_ids = var.ci_plan_mode ? ["subnet-ci00000000000000"] : data.aws_subnets.default[0].ids
  aws_ami_id = var.ci_plan_mode ? "ami-0c7217cdde317cfec" : (
    var.deployment_target == "vm" ? data.aws_ami.ubuntu[0].id : "ami-unused"
  )
  aws_account_id = var.ci_plan_mode ? "123456789012" : (
    var.deployment_target == "serverless" ? data.aws_caller_identity.current[0].account_id : "123456789012"
  )
  aws_eks_token = var.ci_plan_mode ? "ci-dummy-eks-token" : (
    var.deployment_target == "kubernetes" ? data.aws_eks_cluster_auth.eks[0].token : ""
  )

  # Render application .env content for EC2
  env_file_content = <<EOT
SCRAPER_IMAGE=${var.scraper_image}
NOTIFICATION_IMAGE=${var.notification_image}
DATABASE_URL=${var.database_url}
SPRING_DATASOURCE_URL=jdbc:postgresql://${local.db_host}${local.db_path_and_query}
SPRING_DATASOURCE_USERNAME=${local.db_user}
SPRING_DATASOURCE_PASSWORD=${local.db_pass}
KAFKA_BROKERS=${var.kafka_brokers}
KAFKA_TOPIC=mangacdc.public.chapters
KAFKA_USERNAME=${var.kafka_username}
KAFKA_PASSWORD=${var.kafka_password}
CDC_ENABLED=true
DISCORD_WEBHOOK_URL=${var.discord_webhook_url}
SLACK_WEBHOOK_URL=${var.slack_webhook_url}
TELEGRAM_BOT_TOKEN=${var.telegram_bot_token}
TELEGRAM_CHAT_ID=${var.telegram_chat_id}
OBSERVABILITY_MODE=${var.observability_mode}
GRAFANA_CLOUD_PROMETHEUS_URL=${var.grafana_cloud_prometheus_url}
GRAFANA_CLOUD_PROMETHEUS_USER=${var.grafana_cloud_prometheus_user}
GRAFANA_CLOUD_API_KEY=${var.grafana_cloud_api_key}
GRAFANA_CLOUD_STACK_URL=${var.grafana_cloud_stack_url}
EOT
}

# -----------------------------------------------------------------------------
# TARGET: VM Deployment (Docker Compose on AWS EC2)
# -----------------------------------------------------------------------------
resource "aws_security_group" "vm_sg" {
  count       = var.deployment_target == "vm" ? 1 : 0
  name        = "manga-cdc-vm-sg-${var.environment}"
  description = "Security group for manga-cdc VM"
  vpc_id      = local.aws_vpc_id

  # Bound internally to 127.0.0.1; only SSH and Caddy (if using QStash proxy) exposed
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "manga-cdc-sg-${var.environment}"
  }
}

data "aws_ami" "ubuntu" {
  count       = var.ci_plan_mode ? 0 : (var.deployment_target == "vm" ? 1 : 0)
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

resource "aws_instance" "app_ec2" {
  count                  = var.deployment_target == "vm" ? 1 : 0
  ami                    = local.aws_ami_id
  instance_type          = var.instance_type
  subnet_id              = local.aws_subnet_ids[0]
  vpc_security_group_ids = [aws_security_group.vm_sg[0].id]

  user_data = templatefile("${path.module}/../templates/startup.sh.tftpl", {
    env_file_content            = local.env_file_content
    compose_file_content        = file("${path.module}/../../docker-compose.prod.yml")
    caddyfile_content           = fileexists("${path.module}/../../Caddyfile") ? file("${path.module}/../../Caddyfile") : ""
    observability_cloud_enabled = var.observability_mode == "grafana-cloud"
    observability_cloud_content = file("${path.module}/../../docker-compose.observability-cloud.yml")
    alloy_config_content        = file("${path.module}/../../alloy/config.prod.alloy")
    observability_flags         = var.observability_mode == "grafana-cloud" ? "-f docker-compose.observability-cloud.yml" : ""
  })

  user_data_replace_on_change = true

  tags = {
    Name = "manga-cdc-vm-${var.environment}"
    Env  = var.environment
  }
}

# -----------------------------------------------------------------------------
# TARGET: EKS Cluster + Helm Deployment (Kubernetes)
# -----------------------------------------------------------------------------
resource "aws_iam_role" "eks_role" {
  count = var.deployment_target == "kubernetes" ? 1 : 0
  name  = "manga-cdc-eks-cluster-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "eks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "eks_policy" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.eks_role[0].name
}

resource "aws_eks_cluster" "eks" {
  count    = var.deployment_target == "kubernetes" ? 1 : 0
  name     = "manga-cdc-eks-${var.environment}"
  role_arn = aws_iam_role.eks_role[0].arn

  vpc_config {
    subnet_ids = local.aws_subnet_ids
  }

  depends_on = [aws_iam_role_policy_attachment.eks_policy]
}

resource "aws_iam_role" "node_role" {
  count = var.deployment_target == "kubernetes" ? 1 : 0
  name  = "manga-cdc-eks-node-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "node_policy" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.node_role[0].name
}

resource "aws_iam_role_policy_attachment" "cni_policy" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.node_role[0].name
}

resource "aws_iam_role_policy_attachment" "ecr_policy" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.node_role[0].name
}

resource "aws_eks_node_group" "eks_nodes" {
  count           = var.deployment_target == "kubernetes" ? 1 : 0
  cluster_name    = aws_eks_cluster.eks[0].name
  node_group_name = "manga-cdc-node-group"
  node_role_arn   = aws_iam_role.node_role[0].arn
  subnet_ids      = local.aws_subnet_ids
  instance_types  = [var.eks_node_type]

  scaling_config {
    desired_size = var.eks_node_count
    max_size     = var.eks_node_count + 1
    min_size     = 1
  }

  depends_on = [
    aws_iam_role_policy_attachment.node_policy,
    aws_iam_role_policy_attachment.cni_policy,
    aws_iam_role_policy_attachment.ecr_policy
  ]
}

data "aws_eks_cluster_auth" "eks" {
  count = var.ci_plan_mode ? 0 : (var.deployment_target == "kubernetes" ? 1 : 0)
  name  = aws_eks_cluster.eks[0].name
}

provider "kubernetes" {
  host                   = var.deployment_target == "kubernetes" ? aws_eks_cluster.eks[0].endpoint : ""
  cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(aws_eks_cluster.eks[0].certificate_authority[0].data) : ""
  token                  = var.deployment_target == "kubernetes" ? local.aws_eks_token : ""
}

provider "helm" {
  kubernetes {
    host                   = var.deployment_target == "kubernetes" ? aws_eks_cluster.eks[0].endpoint : ""
    cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(aws_eks_cluster.eks[0].certificate_authority[0].data) : ""
    token                  = var.deployment_target == "kubernetes" ? local.aws_eks_token : ""
  }
}

resource "helm_release" "manga_cdc" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  name       = "manga-cdc"
  chart      = "${path.module}/../../helm/manga-cdc"
  namespace  = "default"
  depends_on = [aws_eks_node_group.eks_nodes[0]]

  set {
    name  = "images.scraper"
    value = var.scraper_image
  }

  set {
    name  = "images.notification"
    value = var.notification_image
  }

  set {
    name  = "database.url"
    value = var.database_url
  }

  set {
    name  = "database.jdbcUrl"
    value = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
  }

  set {
    name  = "database.username"
    value = local.db_user
  }

  set {
    name  = "database.password"
    value = local.db_pass
  }

  set {
    name  = "eventing.kafka.enabled"
    value = "true"
  }

  set {
    name  = "eventing.kafka.brokers"
    value = var.kafka_brokers
  }

  set {
    name  = "eventing.kafka.username"
    value = var.kafka_username
  }

  set {
    name  = "eventing.kafka.password"
    value = var.kafka_password
  }

  set {
    name  = "discord.webhookUrl"
    value = var.discord_webhook_url
  }

  set {
    name  = "slack.webhookUrl"
    value = var.slack_webhook_url
  }

  set {
    name  = "telegram.botToken"
    value = var.telegram_bot_token
  }

  set {
    name  = "telegram.chatId"
    value = var.telegram_chat_id
  }

  set {
    name  = "notifiers.discord"
    value = var.discord_webhook_url != "" ? "true" : "false"
  }

  set {
    name  = "notifiers.slack"
    value = var.slack_webhook_url != "" ? "true" : "false"
  }

  set {
    name  = "notifiers.telegram"
    value = var.telegram_bot_token != "" ? "true" : "false"
  }
}

# -----------------------------------------------------------------------------
# TARGET: AWS ECS Fargate + EventBridge Scheduler (Serverless)
# -----------------------------------------------------------------------------
locals {
  aws_fargate_env = [
    for k, v in {
      DATABASE_URL                  = var.database_url
      SPRING_DATASOURCE_URL         = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
      SPRING_DATASOURCE_USERNAME    = local.db_user
      SPRING_DATASOURCE_PASSWORD    = local.db_pass
      KAFKA_BROKERS                 = var.kafka_brokers
      KAFKA_TOPIC                   = "mangacdc.public.chapters"
      KAFKA_USERNAME                = var.kafka_username
      KAFKA_PASSWORD                = var.kafka_password
      CDC_ENABLED                   = "true"
      DISCORD_WEBHOOK_URL           = var.discord_webhook_url
      SLACK_WEBHOOK_URL             = var.slack_webhook_url
      TELEGRAM_BOT_TOKEN            = var.telegram_bot_token
      TELEGRAM_CHAT_ID              = var.telegram_chat_id
      OBSERVABILITY_MODE            = var.observability_mode
      GRAFANA_CLOUD_PROMETHEUS_URL  = var.grafana_cloud_prometheus_url
      GRAFANA_CLOUD_PROMETHEUS_USER = var.grafana_cloud_prometheus_user
      GRAFANA_CLOUD_API_KEY         = var.grafana_cloud_api_key
      GRAFANA_CLOUD_STACK_URL       = var.grafana_cloud_stack_url
    } : { name = k, value = v } if v != ""
  ]
}

resource "aws_ecs_cluster" "ecs" {
  count = var.deployment_target == "serverless" ? 1 : 0
  name  = "manga-cdc-ecs-${var.environment}"
}

resource "aws_iam_role" "ecs_execution_role" {
  count = var.deployment_target == "serverless" ? 1 : 0
  name  = "manga-cdc-ecs-exec-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_exec_policy" {
  count      = var.deployment_target == "serverless" ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.ecs_execution_role[0].name
}

resource "aws_ecs_task_definition" "scraper" {
  count                    = var.deployment_target == "serverless" ? 1 : 0
  family                   = "manga-cdc-scraper"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_execution_role[0].arn

  container_definitions = jsonencode([{
    name      = "scraper"
    image     = var.scraper_image
    essential = true
    environment = concat(local.aws_fargate_env, [{
      name  = "RUN_ONCE"
      value = "true"
    }])
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = "/ecs/manga-cdc-scraper"
        awslogs-region        = var.region
        awslogs-stream-prefix = "scraper"
        awslogs-create-group  = "true"
      }
    }
  }])
}

resource "aws_ecs_task_definition" "notifier" {
  count                    = var.deployment_target == "serverless" ? 1 : 0
  family                   = "manga-cdc-notifier"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_execution_role[0].arn

  container_definitions = jsonencode([{
    name        = "notification-service"
    image       = var.notification_image
    essential   = true
    environment = local.aws_fargate_env
    portMappings = [{
      containerPort = 8080
      hostPort      = 8080
    }]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = "/ecs/manga-cdc-notifier"
        awslogs-region        = var.region
        awslogs-stream-prefix = "notifier"
        awslogs-create-group  = "true"
      }
    }
  }])
}

resource "aws_security_group" "fargate_sg" {
  count       = var.deployment_target == "serverless" ? 1 : 0
  name        = "manga-cdc-fargate-sg-${var.environment}"
  description = "Security group for manga-cdc ECS tasks"
  vpc_id      = local.aws_vpc_id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_ecs_service" "notifier" {
  count           = var.deployment_target == "serverless" ? 1 : 0
  name            = "manga-cdc-notifier"
  cluster         = aws_ecs_cluster.ecs[0].id
  task_definition = aws_ecs_task_definition.notifier[0].arn
  launch_type     = "FARGATE"
  desired_count   = 1

  network_configuration {
    subnets          = local.aws_subnet_ids
    security_groups  = [aws_security_group.fargate_sg[0].id]
    assign_public_ip = true
  }
}

resource "aws_iam_role" "scheduler_role" {
  count = var.deployment_target == "serverless" ? 1 : 0
  name  = "manga-cdc-eb-scheduler-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "scheduler.amazonaws.com" }
    }]
  })
}

resource "aws_iam_policy" "scheduler_policy" {
  count = var.deployment_target == "serverless" ? 1 : 0
  name  = "manga-cdc-eb-scheduler-policy-${var.environment}"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "ecs:RunTask"
        Resource = aws_ecs_task_definition.scraper[0].arn
      },
      {
        Effect   = "Allow"
        Action   = "iam:PassRole"
        Resource = "*"
        Condition = {
          StringLike = {
            "iam:PassedToService" = "ecs-tasks.amazonaws.com"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "scheduler_policy_attach" {
  count      = var.deployment_target == "serverless" ? 1 : 0
  policy_arn = aws_iam_policy.scheduler_policy[0].arn
  role       = aws_iam_role.scheduler_role[0].name
}

resource "aws_scheduler_schedule" "scraper_schedule" {
  count = var.deployment_target == "serverless" ? 1 : 0
  name  = "manga-cdc-scraper-schedule-${var.environment}"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = var.aws_scheduler_schedule

  target {
    arn      = "arn:aws:ecs:${var.region}:${local.aws_account_id}:cluster/${aws_ecs_cluster.ecs[0].name}"
    role_arn = aws_iam_role.scheduler_role[0].arn

    input = jsonencode({
      taskDefinition = aws_ecs_task_definition.scraper[0].arn
      count          = 1
      networkConfiguration = {
        awsvpcConfiguration = {
          assignPublicIp = "ENABLED"
          subnets        = local.aws_subnet_ids
          securityGroups = [aws_security_group.fargate_sg[0].id]
        }
      }
      launchType = "FARGATE"
    })
  }

  depends_on = [
    aws_iam_role_policy_attachment.scheduler_policy_attach
  ]
}

data "aws_caller_identity" "current" {
  count = var.ci_plan_mode ? 0 : (var.deployment_target == "serverless" ? 1 : 0)
}
