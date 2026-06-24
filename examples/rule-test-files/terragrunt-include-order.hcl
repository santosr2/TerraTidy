# Canonical order is include, then dependency, then everything else;
# this file inverts that order to trigger style.terragrunt-include-first.

locals {
  region = "us-east-1"
}

dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = {
  vpc_id = dependency.vpc.outputs.vpc_id
  region = local.region
}
