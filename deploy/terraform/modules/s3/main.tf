variable "name" { type = string }
variable "tags" { type = map(string) }

resource "aws_s3_bucket" "attachments" {
  bucket = "${var.name}-attachments"
  tags   = var.tags
}

resource "aws_s3_bucket" "inbound" {
  bucket = "${var.name}-inbound"
  tags   = var.tags
}

resource "aws_s3_bucket_public_access_block" "attachments" {
  bucket                  = aws_s3_bucket.attachments.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "inbound" {
  bucket                  = aws_s3_bucket.inbound.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "attachments" {
  bucket = aws_s3_bucket.attachments.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "inbound" {
  bucket = aws_s3_bucket.inbound.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_versioning" "attachments" {
  bucket = aws_s3_bucket.attachments.id
  versioning_configuration {
    status = "Enabled"
  }
}

output "bucket" { value = aws_s3_bucket.attachments.id }
output "inbound_bucket" { value = aws_s3_bucket.inbound.id }
output "attachments_arn" { value = aws_s3_bucket.attachments.arn }
output "inbound_arn" { value = aws_s3_bucket.inbound.arn }
