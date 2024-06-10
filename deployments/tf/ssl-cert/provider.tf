# Configure the AWS Provider

# default & current
provider "aws" {
  alias  = "west"
  region = "us-west-2"
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

# for new resources
provider "aws" {
  alias  = "east"
  region = "us-east-1"
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

# configure s3 as the backend and env var 
terraform {
  backend "s3" {
    encrypt = true
    key     = "terraform/prebid-server-eks"
    region  = "us-east-1"
  }
}
