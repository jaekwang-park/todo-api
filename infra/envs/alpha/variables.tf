variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "aws_profile" {
  description = "AWS CLI profile (empty string uses default credentials)"
  type        = string
  default     = ""
}

variable "project" {
  description = "Project name"
  type        = string
}

variable "env" {
  description = "Environment name"
  type        = string
}

# Networking
variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
}

variable "azs" {
  description = "Availability zones"
  type        = list(string)
}

variable "public_subnet_cidrs" {
  description = "Public subnet CIDR blocks"
  type        = list(string)
}

variable "private_subnet_cidrs" {
  description = "Private subnet CIDR blocks"
  type        = list(string)
}

# Database
variable "db_name" {
  description = "Database name"
  type        = string
}

variable "db_user" {
  description = "Database master username"
  type        = string
}

variable "aurora_engine_version" {
  description = "Aurora PostgreSQL engine version"
  type        = string
}

variable "aurora_min_capacity" {
  description = "Aurora serverless v2 minimum ACU"
  type        = number
}

variable "aurora_max_capacity" {
  description = "Aurora serverless v2 maximum ACU"
  type        = number
}

# ECS
variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

variable "ecs_cpu" {
  description = "ECS task CPU units"
  type        = number
}

variable "ecs_memory" {
  description = "ECS task memory (MiB)"
  type        = number
}

variable "ecs_desired_count" {
  description = "Number of ECS tasks"
  type        = number
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
}

# Cognito (passed via TF_VAR_*)
variable "cognito_user_pool_id" {
  description = "Cognito User Pool ID"
  type        = string
}

variable "cognito_app_client_id" {
  description = "Cognito App Client ID"
  type        = string
}

variable "cognito_app_client_secret" {
  description = "Cognito App Client Secret"
  type        = string
  sensitive   = true
}

# GitHub Actions OIDC
variable "github_repository" {
  description = "GitHub repository in format owner/repo"
  type        = string
  default     = ""
}
