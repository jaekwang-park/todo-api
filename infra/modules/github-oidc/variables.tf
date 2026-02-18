variable "project" {
  description = "Project name"
  type        = string
}

variable "env" {
  description = "Environment name"
  type        = string
}

variable "github_repository" {
  description = "GitHub repository in format owner/repo"
  type        = string
}

variable "ecr_repository_arn" {
  description = "ARN of the ECR repository"
  type        = string
}

variable "ecs_cluster_arn" {
  description = "ARN of the ECS cluster"
  type        = string
}

variable "ecs_service_arn" {
  description = "ARN of the ECS service"
  type        = string
}

variable "task_definition_arn_prefix" {
  description = "ARN prefix for ECS task definitions (without revision)"
  type        = string
}

variable "task_execution_role_arn" {
  description = "ARN of the ECS task execution role"
  type        = string
}

variable "task_role_arn" {
  description = "ARN of the ECS task role"
  type        = string
}

variable "tf_state_bucket" {
  description = "S3 bucket for Terraform state"
  type        = string
}

variable "tf_lock_table" {
  description = "DynamoDB table for Terraform state lock"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}
