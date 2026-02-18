resource "aws_db_subnet_group" "main" {
  name       = "${var.project}-${var.env}-db-subnet"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${var.project}-${var.env}-db-subnet"
  }
}

resource "aws_rds_cluster" "main" {
  cluster_identifier = "${var.project}-${var.env}"
  engine             = "aurora-postgresql"
  engine_mode        = "provisioned"
  engine_version     = var.engine_version
  database_name      = var.db_name
  master_username    = var.db_user
  master_password    = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [var.security_group_id]

  storage_encrypted   = true
  deletion_protection = var.deletion_protection
  skip_final_snapshot = var.skip_final_snapshot

  serverlessv2_scaling_configuration {
    min_capacity = var.min_capacity
    max_capacity = var.max_capacity
  }

  tags = {
    Name = "${var.project}-${var.env}"
  }
}

resource "aws_rds_cluster_instance" "main" {
  identifier         = "${var.project}-${var.env}-1"
  cluster_identifier = aws_rds_cluster.main.id
  instance_class     = "db.serverless"
  engine             = aws_rds_cluster.main.engine
  engine_version     = aws_rds_cluster.main.engine_version

  tags = {
    Name = "${var.project}-${var.env}-1"
  }
}
