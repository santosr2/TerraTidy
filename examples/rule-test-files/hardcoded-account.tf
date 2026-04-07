# File with hardcoded AWS account ID - triggers Bash rule "no-hardcoded-account-id"
resource "aws_iam_role" "example" {
  name = "example-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          # Hardcoded account ID below
          AWS = "arn:aws:iam::123456789012:root"
        }
      }
    ]
  })

  tags = {
    Environment = "test"
  }
}
