variable "project" {
  description = "Project name"
  type        = string
}

variable "env" {
  description = "Environment name"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "public_subnet_ids" {
  description = "Public subnet IDs for ECS tasks (assign_public_ip)"
  type        = list(string)
}

variable "security_group_id" {
  description = "Security group ID for ECS tasks"
  type        = string
}

variable "ecr_repository_url" {
  description = "ECR repository URL"
  type        = string
}

variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

variable "task_execution_role_arn" {
  description = "ECS task execution role ARN"
  type        = string
}

variable "task_role_arn" {
  description = "ECS task role ARN"
  type        = string
}

variable "target_group_arn" {
  description = "ALB target group ARN"
  type        = string
}

variable "container_port" {
  description = "Container port"
  type        = number
  default     = 8080
}

variable "cpu" {
  description = "Fargate CPU units"
  type        = number
  default     = 256
}

variable "memory" {
  description = "Fargate memory (MiB)"
  type        = number
  default     = 512
}

variable "desired_count" {
  description = "Number of ECS tasks"
  type        = number
  default     = 1
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 14
}

variable "aws_region" {
  description = "AWS region for CloudWatch logs"
  type        = string
}

variable "env_vars" {
  description = "Environment variables for the container"
  type        = map(string)
  default     = {}
}

variable "secret_vars" {
  description = "Secret variables (name -> SSM parameter ARN)"
  type        = map(string)
  default     = {}
}

variable "enable_execute_command" {
  description = "Enable ECS Exec for the service"
  type        = bool
  default     = false
}
