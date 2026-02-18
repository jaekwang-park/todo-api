output "cluster_endpoint" {
  description = "Aurora cluster writer endpoint"
  value       = aws_rds_cluster.main.endpoint
}

output "cluster_port" {
  description = "Aurora cluster port"
  value       = aws_rds_cluster.main.port
}

output "db_name" {
  description = "Database name"
  value       = aws_rds_cluster.main.database_name
}
