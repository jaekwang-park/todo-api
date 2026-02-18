# Alpha environment configuration
# Secrets (cognito_user_pool_id, cognito_app_client_id, cognito_app_client_secret)
# are passed via TF_VAR_* environment variables.

aws_region  = "ap-northeast-1"
aws_profile = "todo-alpha"
project     = "todo-api"
env         = "alpha"

# Networking
vpc_cidr             = "10.0.0.0/16"
azs                  = ["ap-northeast-1a", "ap-northeast-1c"]
public_subnet_cidrs  = ["10.0.1.0/24", "10.0.2.0/24"]
private_subnet_cidrs = ["10.0.101.0/24", "10.0.102.0/24"]

# Database
db_name               = "todo"
db_user               = "todo"
aurora_engine_version = "15.8"
aurora_min_capacity   = 0.5
aurora_max_capacity   = 1.0

# ECS
ecs_cpu            = 256
ecs_memory         = 512
ecs_desired_count  = 1
log_retention_days = 14
