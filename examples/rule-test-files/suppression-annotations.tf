# Examples of suppression annotations for TerraTidy
# These annotations suppress findings from style, lint, and policy engines

# File-level suppression - suppresses require-description for entire file
# terratidy:ignore-file:require-description

# Block-level suppression - suppresses the next resource block only
# terratidy:ignore:require-tags
resource "aws_instance" "suppressed_tags" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  # This resource won't trigger require-tags due to the annotation above
}

# Inline suppression - suppresses on the same line
resource "aws_s3_bucket" "example" { # terratidy:ignore:naming-convention
  bucket = "MyBucketWithBadName"
}

# Wildcard suppression - suppresses all style rules for next block
# terratidy:ignore:style.*
resource "aws_instance" "no_style_checks" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}

# Multiple annotations can be stacked
# terratidy:ignore:require-tags
# terratidy:ignore:naming-convention
resource "aws_instance" "multiple_suppressions" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}

# Resource without suppressions - will trigger findings normally
resource "aws_instance" "no_suppression" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  # This will trigger require-tags and require-description
}
