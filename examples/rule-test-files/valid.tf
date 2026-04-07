# Compliant file - should not trigger any rules
variable "account_id" {
  description = "AWS account ID"
  type        = string
}

resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  description   = "Web server instance"

  tags = {
    Name        = "web-server"
    Environment = "production"
  }
}

resource "aws_iam_role" "example" {
  name        = "example-role"
  description = "Example IAM role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${var.account_id}:root"
        }
      }
    ]
  })

  tags = {
    Environment = "production"
  }
}
