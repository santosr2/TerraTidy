package format

import (
	"testing"
)

func FuzzFormat(f *testing.F) {
	f.Add([]byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  tags = {
    Name = "test"
  }
}
`))
	f.Add([]byte(`variable "name" {
  type    = string
  default = "hello"
}
`))
	f.Add([]byte(`terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`))
	f.Add([]byte(``))
	f.Add([]byte(`{`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = Format(data)
	})
}
