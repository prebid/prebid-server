# Find the environment-specific subdomain zone of adthrive.com
data "aws_route53_zone" "env_adthrive_com" {
  name         = "${var.environment}.adthrive.com."
  private_zone = false
}

# Find the ALB for the "prebid-deployment"
data "aws_lb" "prebid_deployment" {
  tags = {
    environment     = var.environment
    kube_deployment = "prebid-deployment"
  }
  provider = aws.west
}

# Find the ALB for the "prebid-deployment-west"
data "aws_lb" "prebid_deployment_west_alb" {
  tags = {
    "ingress.k8s.aws/resource" = "LoadBalancer"
    environment                = var.environment
    kube_deployment            = "prebid-deployment-west"
  }
  provider = aws.west
}

# Find the NLB for the "prebid-deployment-west"
data "aws_lb" "prebid_deployment_west_nlb" {
  tags = {
    "service.k8s.aws/resource" = "LoadBalancer"
    environment                = var.environment
    kube_deployment            = "prebid-deployment-west"
  }
  provider = aws.west
}


# Find the ALB for the "prebid-deployment-east"
data "aws_lb" "prebid_deployment_east" {
  tags = {
    environment     = var.environment
    kube_deployment = "prebid-deployment-east"
  }
  provider = aws.east
}

data "aws_lb" "prebid_proxy" {
  tags = {
    environment = var.environment
    service     = "prebid-server-proxy"
  }
  provider = aws.west
}

# Create the necessary records
module "env_adthrive_com" {
  source = "terraform-aws-modules/route53/aws//modules/records"

  zone_id = data.aws_route53_zone.env_adthrive_com.id

  records = [
    {
      name    = "prebid"
      type    = "CNAME"
      ttl     = 300
      records = [data.aws_lb.prebid_deployment.dns_name]
    },
    {
      name = "prebid-west"
      type = "A"
      alias = {
        name    = data.aws_lb.prebid_proxy.dns_name
        zone_id = data.aws_lb.prebid_proxy.zone_id
      }
    },
    {
      name    = "prebid-west-java"
      type    = "CNAME"
      ttl     = 300
      records = [data.aws_lb.prebid_deployment_west_alb.dns_name]
    },
    {
      name = "prebid-west-internal"
      type = "A"
      alias = {
        name    = data.aws_lb.prebid_deployment_west_nlb.dns_name
        zone_id = data.aws_lb.prebid_deployment_west_nlb.zone_id
      }
    },
    {
      name    = "prebid-east"
      type    = "CNAME"
      ttl     = 300
      records = [data.aws_lb.prebid_deployment_east.dns_name]
    },
    {
      name    = "prebid-proxy"
      type    = "CNAME"
      ttl     = 300
      records = [data.aws_lb.prebid_proxy.dns_name]
    }
  ]
}
