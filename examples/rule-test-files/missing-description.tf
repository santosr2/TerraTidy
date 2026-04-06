# Resource without description - triggers YAML rule "require-description"
resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
  # Missing description attribute

  tags = {
    Environment = "production"
  }
}
