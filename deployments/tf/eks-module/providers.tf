terraform {

  required_version = "~> 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.47"
    }
  }

  backend "s3" {
    key    = "terraform/prebid-server-platform" # Test command line override
    region = "us-east-1"
  }
}

# Provider for deployment region
provider "aws" {
  region = var.region
  default_tags {
    tags = {
      environment        = var.environment
      owner              = "slickstream"
      provisioner        = "terraform"
      provisioner-source = "@slickstream/slickstream-platform"
      # provisioner-commit = var.commit_sha
      map-migrated = 1
    }
  }
}

# Provider for execution region (regional API endpoints)
provider "aws" {
  region = "us-east-1"
  alias  = "east"
  default_tags {
    tags = {
      owner              = "ad-code-delivery"
      service            = "prebid-server"
      environment        = var.environment
      provisioner        = "terraform"
      provisioner-source = "@cafemedia/Prebid-Server-EKS"
    }
  }
}

# Provider for execution region (regional API endpoints)
provider "aws" {
  region = "us-west-2"
  alias  = "west"
  default_tags {
    tags = {
      owner              = "ad-code-delivery"
      service            = "prebid-server"
      environment        = var.environment
      provisioner        = "terraform"
      provisioner-source = "@cafemedia/Prebid-Server-EKS"
    }
  }
}
