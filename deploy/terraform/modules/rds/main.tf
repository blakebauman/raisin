variable "name" { type = string }
variable "vpc_id" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_ids" { type = list(string) }
variable "tags" { type = map(string) }
variable "db_password" {
  type      = string
  sensitive = true
}
variable "instance_class" {
  type    = string
  default = "db.t4g.medium"
}

resource "aws_db_subnet_group" "main" {
  name       = var.name
  subnet_ids = var.subnet_ids
  tags       = var.tags
}

resource "aws_db_instance" "main" {
  identifier                 = var.name
  engine                     = "postgres"
  engine_version             = "16"
  instance_class             = var.instance_class
  allocated_storage          = 50
  max_allocated_storage      = 200
  storage_type               = "gp3"
  storage_encrypted          = true
  db_name                    = "raisin"
  username                   = "raisin"
  password                   = var.db_password
  db_subnet_group_name       = aws_db_subnet_group.main.name
  vpc_security_group_ids     = var.security_group_ids
  skip_final_snapshot        = true
  deletion_protection        = false
  publicly_accessible        = false
  backup_retention_period    = 7
  auto_minor_version_upgrade = true
  tags                       = var.tags
}

output "endpoint" { value = aws_db_instance.main.endpoint }
output "address" { value = aws_db_instance.main.address }
output "port" { value = aws_db_instance.main.port }
