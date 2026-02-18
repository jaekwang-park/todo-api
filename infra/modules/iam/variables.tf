variable "project" {
  description = "Project name"
  type        = string
}

variable "env" {
  description = "Environment name"
  type        = string
}

variable "ecr_repository_arn" {
  description = "ARN of the ECR repository"
  type        = string
}

variable "ssm_parameter_arns" {
  description = "ARNs of SSM parameters that ECS task execution role can read"
  type        = list(string)
}

variable "enable_ecs_exec" {
  description = "Add SSM session permissions to task role for ECS Exec"
  type        = bool
  default     = false
}
