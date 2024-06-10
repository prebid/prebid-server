locals {
  environment      = var.environment
  project_name     = var.project_name
  datadog_api_key  = var.datadog_api_key
  datadog_app_key  = var.datadog_app_key
  workspace_suffix = terraform.workspace == "default" ? "" : "-${terraform.workspace}"
}

# New cluster with bigger subnets for more IPs to avoid exhaustion issues
module "prebid-eks-usw2-v2" {
  source      = "git::https://github.com/cafemedia/terraform-module-eks.git?ref=0.1.7"
  region      = "us-west-2"
  environment = local.environment

  providers = {
    aws = aws.west
  }

  cluster_name       = "${local.project_name}-usw2${local.workspace_suffix}-v2"
  datadog_api_key    = local.datadog_api_key
  datadog_app_key    = local.datadog_app_key
  enable_nat_gateway = true
  # helm_doit_deployment_id   = data.aws_ssm_parameter.doit_deployment_id_west.value
  helm_karpenter_version    = "v0.31.3"
  helm_observe_cluster_name = "${local.project_name}-usw2${local.workspace_suffix}-v2-${local.environment}"
  helm_observe_token        = data.aws_ssm_parameter.observe_token_west.value
  kubernetes_version        = "1.28"
  map_public_ip_on_launch   = true
  service                   = local.project_name
  single_nat_gateway        = false
  additional_tags_for_eks_provisioned_resources = {
    owner = "ad-code-delivery"
  }
  use_private_subnets_for_eks             = true
  use_private_subnets_for_karpenter_nodes = false
  use_private_subnets_for_managed_nodes   = true
  use_public_subnets_for_eks              = true
  use_public_subnets_for_karpenter_nodes  = true
  use_public_subnets_for_managed_nodes    = false
}

# New cluster with bigger subnets for more IPs to avoid exhaustion issues
module "prebid-eks-use1-v2" {
  source      = "git::https://github.com/cafemedia/terraform-module-eks.git?ref=0.1.7"
  region      = "us-east-1"
  environment = local.environment

  providers = {
    aws = aws.east
  }

  cluster_name           = "${local.project_name}-use1${local.workspace_suffix}-v2"
  datadog_api_key        = local.datadog_api_key
  datadog_app_key        = local.datadog_app_key
  enable_nat_gateway     = true
  helm_karpenter_version = "v0.31.3"
  # helm_doit_deployment_id   = data.aws_ssm_parameter.doit_deployment_id_east.value
  helm_observe_cluster_name = "${local.project_name}-use1${local.workspace_suffix}-v2-${local.environment}"
  helm_observe_token        = data.aws_ssm_parameter.observe_token_east.value
  kubernetes_version        = "1.28"
  map_public_ip_on_launch   = true
  service                   = local.project_name
  single_nat_gateway        = false
  additional_tags_for_eks_provisioned_resources = {
    owner = "ad-code-delivery"
  }
  use_private_subnets_for_eks             = true
  use_private_subnets_for_karpenter_nodes = false
  use_private_subnets_for_managed_nodes   = true
  use_public_subnets_for_eks              = true
  use_public_subnets_for_karpenter_nodes  = true
  use_public_subnets_for_managed_nodes    = false
}
