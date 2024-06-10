#
# Observe-related data lookups
#

data "aws_ssm_parameter" "observe_token_west" {
  name     = "/${local.environment}/eks/${local.project_name}-usw2${local.workspace_suffix}-v2/observe_token"
  provider = aws.west
}

data "aws_ssm_parameter" "observe_token_east" {
  name     = "/${local.environment}/eks/${local.project_name}-use1${local.workspace_suffix}-v2/observe_token"
  provider = aws.east
}

#
# DoIT-related data lookups
#

data "aws_ssm_parameter" "doit_deployment_id_west" {
  name     = "/${local.environment}/eks/${local.project_name}-usw2${local.workspace_suffix}-v2/doit_deployment_id"
  provider = aws.west
}

data "aws_ssm_parameter" "doit_deployment_id_east" {
  name     = "/${local.environment}/eks/${local.project_name}-use1${local.workspace_suffix}-v2/doit_deployment_id"
  provider = aws.east
}
