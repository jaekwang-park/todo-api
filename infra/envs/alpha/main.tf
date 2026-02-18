terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }

  backend "s3" {
    bucket         = "todo-alpha-tf-state-jaekwang"
    key            = "alpha/terraform.tfstate"
    region         = "ap-northeast-1"
    dynamodb_table = "todo-alpha-tf-lock"
    encrypt        = true
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile != "" ? var.aws_profile : null

  default_tags {
    tags = {
      Project     = var.project
      Environment = var.env
      ManagedBy   = "terraform"
    }
  }
}

# ==============================================================================
# Independent modules
# ==============================================================================

module "networking" {
  source = "../../modules/networking"

  project              = var.project
  env                  = var.env
  vpc_cidr             = var.vpc_cidr
  azs                  = var.azs
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
}

module "ecr" {
  source = "../../modules/ecr"

  project = var.project
  env     = var.env
}

# ==============================================================================
# Security Groups (created at env level to avoid circular dependencies)
# ==============================================================================

resource "aws_security_group" "alb" {
  name        = "${var.project}-${var.env}-alb-sg"
  description = "ALB security group"
  vpc_id      = module.networking.vpc_id

  ingress {
    description = "HTTP from internet"
    from_port   = 80
    to_port     = 80
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
    Name = "${var.project}-${var.env}-alb-sg"
  }
}

resource "aws_security_group" "ecs" {
  name        = "${var.project}-${var.env}-ecs-sg"
  description = "ECS tasks security group"
  vpc_id      = module.networking.vpc_id

  ingress {
    description     = "From ALB"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project}-${var.env}-ecs-sg"
  }
}

resource "aws_security_group" "rds" {
  name        = "${var.project}-${var.env}-rds-sg"
  description = "RDS security group"
  vpc_id      = module.networking.vpc_id

  ingress {
    description     = "PostgreSQL from ECS"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project}-${var.env}-rds-sg"
  }
}

# ==============================================================================
# Secrets (SSM Parameter Store)
# ==============================================================================

resource "random_password" "db_password" {
  length  = 32
  special = false
}

resource "aws_ssm_parameter" "db_password" {
  name  = "/${var.project}/${var.env}/db-password"
  type  = "SecureString"
  value = random_password.db_password.result

  tags = {
    Name = "${var.project}-${var.env}-db-password"
  }
}

resource "aws_ssm_parameter" "cognito_app_client_secret" {
  name  = "/${var.project}/${var.env}/cognito-app-client-secret"
  type  = "SecureString"
  value = var.cognito_app_client_secret

  tags = {
    Name = "${var.project}-${var.env}-cognito-app-client-secret"
  }
}

# ==============================================================================
# IAM
# ==============================================================================

module "iam" {
  source = "../../modules/iam"

  project            = var.project
  env                = var.env
  ecr_repository_arn = module.ecr.repository_arn
  ssm_parameter_arns = [
    aws_ssm_parameter.db_password.arn,
    aws_ssm_parameter.cognito_app_client_secret.arn,
  ]
  enable_ecs_exec = true
}

# ==============================================================================
# ALB
# ==============================================================================

module "alb" {
  source = "../../modules/alb"

  project           = var.project
  env               = var.env
  vpc_id            = module.networking.vpc_id
  public_subnet_ids = module.networking.public_subnet_ids
  security_group_id = aws_security_group.alb.id
  health_check_path = "/health"
  container_port    = 8080
}

# ==============================================================================
# RDS (Aurora Serverless v2)
# ==============================================================================

module "rds" {
  source = "../../modules/rds"

  project             = var.project
  env                 = var.env
  private_subnet_ids  = module.networking.private_subnet_ids
  security_group_id   = aws_security_group.rds.id
  db_name             = var.db_name
  db_user             = var.db_user
  db_password         = random_password.db_password.result
  engine_version      = var.aurora_engine_version
  min_capacity        = var.aurora_min_capacity
  max_capacity        = var.aurora_max_capacity
  deletion_protection = false
  skip_final_snapshot = true
}

# ==============================================================================
# ECS
# ==============================================================================

module "ecs" {
  source = "../../modules/ecs"

  project                 = var.project
  env                     = var.env
  vpc_id                  = module.networking.vpc_id
  public_subnet_ids       = module.networking.public_subnet_ids
  security_group_id       = aws_security_group.ecs.id
  ecr_repository_url      = module.ecr.repository_url
  image_tag               = var.image_tag
  task_execution_role_arn = module.iam.ecs_task_execution_role_arn
  task_role_arn           = module.iam.ecs_task_role_arn
  target_group_arn        = module.alb.target_group_arn
  container_port          = 8080
  cpu                     = var.ecs_cpu
  memory                  = var.ecs_memory
  desired_count           = var.ecs_desired_count
  log_retention_days      = var.log_retention_days
  aws_region              = var.aws_region

  enable_execute_command = true

  env_vars = {
    APP_ENV               = var.env
    SERVER_PORT           = "8080"
    DB_HOST               = module.rds.cluster_endpoint
    DB_PORT               = tostring(module.rds.cluster_port)
    DB_USER               = var.db_user
    DB_NAME               = var.db_name
    DB_SSLMODE            = "require"
    LOG_LEVEL             = "info"
    AUTH_DEV_MODE         = "false"
    COGNITO_REGION        = var.aws_region
    COGNITO_USER_POOL_ID  = var.cognito_user_pool_id
    COGNITO_APP_CLIENT_ID = var.cognito_app_client_id
  }

  secret_vars = {
    DB_PASSWORD               = aws_ssm_parameter.db_password.arn
    COGNITO_APP_CLIENT_SECRET = aws_ssm_parameter.cognito_app_client_secret.arn
  }
}

# ==============================================================================
# GitHub Actions OIDC
# ==============================================================================

module "github_oidc" {
  source = "../../modules/github-oidc"
  count  = var.github_repository != "" ? 1 : 0

  project                    = var.project
  env                        = var.env
  github_repository          = var.github_repository
  ecr_repository_arn         = module.ecr.repository_arn
  ecs_cluster_arn            = module.ecs.cluster_arn
  ecs_service_arn            = module.ecs.service_arn
  task_definition_arn_prefix = module.ecs.task_definition_arn
  task_execution_role_arn    = module.iam.ecs_task_execution_role_arn
  task_role_arn              = module.iam.ecs_task_role_arn
  tf_state_bucket            = "todo-alpha-tf-state-jaekwang"
  tf_lock_table              = "todo-alpha-tf-lock"
  aws_region                 = var.aws_region
}
