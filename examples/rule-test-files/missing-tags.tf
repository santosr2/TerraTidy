# Resource without tags - triggers Go rule "require-tags"
# Note: Also triggers require-description YAML rule
resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  # Missing tags attribute
}
