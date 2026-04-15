package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/internal/config"
)

// Sample Terraform configurations of varying complexity for benchmarks
var (
	simpleConfig = `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`

	mediumConfig = `terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 2
}

resource "aws_instance" "web" {
  count = var.instance_count

  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name        = "web-${count.index}"
    Environment = "production"
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "instance_ids" {
  description = "List of instance IDs"
  value       = aws_instance.web[*].id
}
`

	complexConfig = `terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

provider "aws" {
  alias  = "east"
  region = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "Terraform"
    Project     = "benchmark"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(local.common_tags, {
    Name = "${var.environment}-vpc"
  })
}

resource "aws_subnet" "public" {
  count = 3

  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = merge(local.common_tags, {
    Name = "${var.environment}-public-${count.index}"
    Type = "public"
  })
}

resource "aws_subnet" "private" {
  count = 3

  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 10)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = merge(local.common_tags, {
    Name = "${var.environment}-private-${count.index}"
    Type = "private"
  })
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name = "${var.environment}-igw"
  })
}

resource "aws_security_group" "web" {
  name        = "${var.environment}-web-sg"
  description = "Security group for web servers"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.environment}-web-sg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_instance" "web" {
  for_each = toset(["web1", "web2", "web3"])

  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.public[0].id
  vpc_security_group_ids = [aws_security_group.web.id]

  tags = merge(local.common_tags, {
    Name = "${var.environment}-${each.key}"
    Role = "web"
  })

  lifecycle {
    create_before_destroy = true
  }
}

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 5.0"

  identifier = "${var.environment}-db"

  engine               = "mysql"
  engine_version       = "8.0"
  family               = "mysql8.0"
  major_engine_version = "8.0"
  instance_class       = "db.t3.micro"

  allocated_storage = 20

  db_name  = "mydb"
  username = "admin"
  port     = 3306

  vpc_security_group_ids = [aws_security_group.web.id]
  subnet_ids             = aws_subnet.private[*].id

  tags = local.common_tags
}

output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = aws_subnet.private[*].id
}

output "web_instance_ids" {
  description = "Web instance IDs"
  value       = [for instance in aws_instance.web : instance.id]
}
`

	// deeplyNestedConfig contains HCL with 10+ levels of structural nesting
	// (locals path reaches 11 levels) plus expression-level nesting in jsonencode
	// calls. Used to stress test parsers and style checkers on deeply nested structures.
	deeplyNestedConfig = `# Deeply nested Terraform configuration for benchmark testing
# This tests parser and style engine performance with deep nesting

resource "aws_lambda_function" "deeply_nested" {
  function_name = "deeply-nested-lambda"
  role          = aws_iam_role.lambda.arn
  handler       = "index.handler"
  runtime       = "nodejs18.x"

  environment {
    variables = {
      CONFIG = jsonencode({
        level1 = {
          level2 = {
            level3 = {
              level4 = {
                level5 = {
                  level6 = {
                    level7 = {
                      level8 = {
                        level9 = {
                          level10 = {
                            level11 = {
                              level12 = {
                                value = "deeply nested value"
                                items = ["a", "b", "c"]
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      })
    }
  }

  tags = {
    Name        = "deeply-nested"
    Environment = "benchmark"
  }
}

resource "aws_iam_role" "lambda" {
  name = "deeply-nested-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      Condition = {
        StringEquals = {
          "aws:RequestTag/Project" = {
            nested = {
              deeply = {
                value = ["benchmark"]
              }
            }
          }
        }
      }
    }]
  })
}

locals {
  nested_config = {
    environments = {
      production = {
        settings = {
          database = {
            connection = {
              pool = {
                min = 5
                max = 100
                timeout = {
                  connect = 30
                  idle    = 60
                  retry = {
                    attempts = 3
                    backoff = {
                      initial = 100
                      max     = 5000
                      multiplier = 2
                    }
                  }
                }
              }
            }
          }
        }
      }
      staging = {
        settings = {
          database = {
            connection = {
              pool = {
                min = 2
                max = 20
              }
            }
          }
        }
      }
    }
  }
}

variable "complex_object" {
  type = object({
    level1 = object({
      level2 = object({
        level3 = object({
          level4 = object({
            level5 = object({
              level6 = object({
                level7 = object({
                  level8 = object({
                    value = string
                  })
                })
              })
            })
          })
        })
      })
    })
  })
  default = {
    level1 = {
      level2 = {
        level3 = {
          level4 = {
            level5 = {
              level6 = {
                level7 = {
                  level8 = {
                    value = "default"
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}

output "nested_value" {
  value = var.complex_object.level1.level2.level3.level4.level5.level6.level7.level8.value
}
`
)

func BenchmarkStyleEngine_SimpleConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(simpleConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_MediumConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(mediumConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_ComplexConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(complexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_MultipleFiles(b *testing.B) {
	tmpDir := b.TempDir()

	// Create multiple files
	files := []struct {
		name    string
		content string
	}{
		{"main.tf", complexConfig},
		{"variables.tf", `variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "env" {
  description = "Environment name"
  type        = string
}
`},
		{"outputs.tf", `output "region" {
  description = "AWS region"
  value       = var.region
}

output "env" {
  description = "Environment name"
  value       = var.env
}
`},
	}

	var filePaths []string
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
			b.Fatalf("failed to create file %s: %v", f.name, err)
		}
		filePaths = append(filePaths, path)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, filePaths)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_WithFixMode(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	// Config with style issues that need fixing
	configWithIssues := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`
	if err := os.WriteFile(tmpFile, []byte(configWithIssues), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Re-write the file for each iteration since fix mode modifies it
		if err := os.WriteFile(tmpFile, []byte(configWithIssues), 0o644); err != nil {
			b.Fatalf("failed to re-create temp file: %v", err)
		}
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_SingleRuleEnabled(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(complexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	// Only enable one rule - blank-line-between-blocks
	engine := New(&Config{
		Rules: map[string]RuleConfig{
			"style.blank-line-between-blocks": {Enabled: config.BoolPtr(true)},
			// All other rules will use default enabled/disabled state
		},
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkStyleEngine_DeepNesting benchmarks the style engine against deeply nested HCL structures.
func BenchmarkStyleEngine_DeepNesting(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "deeply_nested.tf")
	if err := os.WriteFile(tmpFile, []byte(deeplyNestedConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkStyleEngine_AllRulesEnabled(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(complexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	// Enable all rules explicitly
	allRules := map[string]RuleConfig{
		"style.blank-line-between-blocks":       {Enabled: config.BoolPtr(true)},
		"style.no-leading-trailing-blank-lines": {Enabled: config.BoolPtr(true)},
		"style.no-empty-blocks":                 {Enabled: config.BoolPtr(true)},
		"style.block-label-case":                {Enabled: config.BoolPtr(true)},
		"style.variable-naming":                 {Enabled: config.BoolPtr(true)},
		"style.output-naming":                   {Enabled: config.BoolPtr(true)},
		"style.local-naming":                    {Enabled: config.BoolPtr(true)},
		"style.for-each-count-first":            {Enabled: config.BoolPtr(true)},
		"style.lifecycle-at-end":                {Enabled: config.BoolPtr(true)},
		"style.tags-at-end":                     {Enabled: config.BoolPtr(true)},
		"style.depends-on-order":                {Enabled: config.BoolPtr(true)},
		"style.source-version-grouped":          {Enabled: config.BoolPtr(true)},
		"style.variable-order":                  {Enabled: config.BoolPtr(true)},
		"style.output-order":                    {Enabled: config.BoolPtr(true)},
		"style.terraform-block-first":           {Enabled: config.BoolPtr(true)},
		"style.provider-block-order":            {Enabled: config.BoolPtr(true)},
		"style.attribute-group-spacing":         {Enabled: config.BoolPtr(true)},
		"style.variables-in-file":               {Enabled: config.BoolPtr(true)},
		"style.outputs-in-file":                 {Enabled: config.BoolPtr(true)},
		"style.providers-in-file":               {Enabled: config.BoolPtr(true)},
		"style.scoped-file-organization":        {Enabled: config.BoolPtr(true)},
		"style.terraform-files-structure":       {Enabled: config.BoolPtr(true)},
		"style.resource-name-matches-type":      {Enabled: config.BoolPtr(true)},
		"style.output-prefix":                   {Enabled: config.BoolPtr(true)},
		"style.module-name-convention":          {Enabled: config.BoolPtr(true)},
		"style.meta-arguments-order":            {Enabled: config.BoolPtr(true)},
		"style.lifecycle-attribute-order":       {Enabled: config.BoolPtr(true)},
		"style.nested-block-order":              {Enabled: config.BoolPtr(true)},
		"style.one-line-attribute-spacing":      {Enabled: config.BoolPtr(true)},
		"style.comment-syntax":                  {Enabled: config.BoolPtr(true)},
		"style.no-trailing-whitespace":          {Enabled: config.BoolPtr(true)},
		"style.consistent-quotes":               {Enabled: config.BoolPtr(true)},
		"style.no-consecutive-blank-lines":      {Enabled: config.BoolPtr(true)},
	}

	engine := New(&Config{
		Rules: allRules,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}
