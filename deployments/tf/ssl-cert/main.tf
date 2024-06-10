# Modern ACM certs provisioned by DNS automatically
# Use this pattern for new certs

module "prebid_server_java_east" {
  source = "terraform-aws-modules/acm/aws"

  providers = {
    aws = aws.east
  }

  domain_name = "prebid.east.${var.environment}.adthrive.com"
  zone_id     = data.aws_route53_zone.env_adthrive_com.id

  validation_method = "DNS"

  create_route53_records = true
}

module "prebid_server_proxy_east" {
  source = "terraform-aws-modules/acm/aws"

  providers = {
    aws = aws.east
  }

  domain_name = "prebid-proxy.east.${var.environment}.adthrive.com"
  zone_id     = data.aws_route53_zone.env_adthrive_com.id

  validation_method = "DNS"

  create_route53_records = true
}

module "prebid_server_java_west" {
  source = "terraform-aws-modules/acm/aws"

  providers = {
    aws = aws.west
  }

  domain_name = "prebid.west.${var.environment}.adthrive.com"
  zone_id     = data.aws_route53_zone.env_adthrive_com.id

  validation_method = "DNS"

  create_route53_records = true
}

module "prebid_server_proxy_west" {
  source = "terraform-aws-modules/acm/aws"

  providers = {
    aws = aws.west
  }

  domain_name = "prebid-proxy.west.${var.environment}.adthrive.com"
  zone_id     = data.aws_route53_zone.env_adthrive_com.id

  validation_method = "DNS"

  create_route53_records = true
}

# Legacy ACM Certs that were provisioned with manual validation
# Avoid using this pattern going forward

resource "aws_acm_certificate" "cert" {
  domain_name       = format("prebid.%s.adthrive.com", var.environment)
  provider          = aws.west
  validation_method = "DNS"

  tags = {
    environment        = var.environment
    owner              = "ACO"
    provisioner        = "terraform"
    provisioner-source = "@cafemedia/Prebid-Server-EKS"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" "cert_use1" {
  domain_name       = format("prebid-east.%s.adthrive.com", var.environment)
  provider          = aws.east
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" "cert_experiment" {
  domain_name       = format("prebid-experiment.%s.adthrive.com", var.environment)
  provider          = aws.west
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" "cert_experiment_use1" {
  domain_name       = format("prebid-experiment-east.%s.adthrive.com", var.environment)
  provider          = aws.east
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" "main_cert_use1" {
  domain_name       = format("prebid.%s.adthrive.com", var.environment)
  provider          = aws.east
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" "cert-west" {
  domain_name       = format("prebid-west.%s.adthrive.com", var.environment)
  validation_method = "DNS"

  tags = {
    environment        = var.environment
    owner              = "ACO"
    provisioner        = "terraform"
    provisioner-source = "@cafemedia/Prebid-Server-EKS"
  }

  lifecycle {
    create_before_destroy = true
  }
}