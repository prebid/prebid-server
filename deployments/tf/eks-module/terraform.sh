#! /bin/bash

# ./terraform.sh <path-to-terraform-files>
#     -t init and plan only
#     -e environment name (development, stage, production)
#     -p aws profile name (in .aws/credentials)
#     -r aws region to create resources inside (us-east-1)
#     -v <key=value> extra variables to define in terraform

# Dependencies: terraform, aws
# @todo: check versions
dependencies=("terraform" "aws")
for dep in ${dependencies[@]}; do
  command -v ${dep} >/dev/null 2>&1 || echo "${dep} is required"
done

# Initial settings
planOnly="false"
destroy="false"
environment="null"
awsProfileName=
awsRegion="us-east-1"
terraformDir=
terraformVariables=()
setWorkspace="false"
terraformWorkspace=default

usage() {
    echo "Usage: $0 [-t plan only ] [ -v key=value... ] [ -m -t directory ] [-f force update] directory" 1>&2
    echo "    -e environment name (development, stage, production)" 1>&2
    echo "    -p aws profile name (in .aws/credentials)" 1>&2
    echo "    -r aws region to create resources inside (us-east-1)" 1>&2
    echo "    -t init and plan only" 1>&2
    echo "    -v <key=value> extra variables to define in terraform" 1>&2
    exit
}

while getopts "e:p:r:w:v:dtcdh" opt; do
  case ${opt} in
    e ) # @todo: remove and use profile to indicate account connection
        # @todo: if needed environment can be passed directly to Terraform
        # with -v
        environment=${OPTARG}
        ;;
    p ) # AWS profile credentials
        awsProfileName=${OPTARG}
        ;;
    r ) # AWS region to connect to
        awsRegion=${OPTARG}
        ;;
    w ) # Don't apply terraform, only init and plan
        setWorkspace="true"
        workspaceName=${OPTARG}
        ;;
    t ) # Don't apply terraform, only init and plan
        planOnly="true"
        ;;
    d ) # Destroy all resources in workspace, only init and destroy
        destroy="true"
        ;;
    v ) # Variables to pass directly to Terraform
        terraformVariables+=($OPTARG)
        ;;
    d ) # Cheap and dirty debugging
        set -x
        ;;
    c ) # Don't cleant terraform plan output
        cleanUp="false"
        ;;
    h )
        usage
        ;;
    * )
        usage
        ;;
    \? )
        usage
        ;;
  esac
done
shift $((OPTIND-1))

# Positional arguments
# Set working directory. "." is default
terraformDir=${1:-.}

# Bail if no profile specified
if [[ -z "$awsProfileName" ]]; then
    echo "AWS account profile must be set"
    exit 1
fi

export AWS_PROFILE=$awsProfileName

# Extract terraform variables to use during invocation
parsedTerraformVariables=
for var in "${terraformVariables[@]}"; do
    parsedTerraformVariables+="-var $var "
done

# Create temp directory file to store terraform plan data
tempDir=$(mktemp -d)
tempPlan=$(mktemp)
export TF_DATA_DIR=$tempDir

# Move to working directory
# @todo use terraform's -chdir option instead
cd $terraformDir

function getAwsAccountId() {
    aws --profile $awsProfileName sts get-caller-identity \
        --query Account \
        --output text
}

function terraformInit() {
    terraform init \
        -reconfigure \
        -var "profile=$awsProfileName" \
        -var "region=$awsRegion" \
        -backend-config="profile=$awsProfileName" \
        -backend-config="bucket=iac-state-$awsAccountId" \
        -backend-config="region=us-east-1"
}

function terraformCreateWorkspace() {
    terraform workspace select --or-create $workspaceName
}

function terraformPlan() {
    terraform plan \
        -var "profile=$awsProfileName" \
        -var "environment=$environment" \
        $parsedTerraformVariables \
        -out $tempPlan
}

function terraformApply() {
    terraform apply $tempPlan
}

function terraformDestroy() {
    # echo "terraform destroy called"
    terraform apply -destroy \
        -var "profile=$awsProfileName" \
        -var "environment=$environment" \
        $parsedTerraformVariables
}

function cleanUp() {
    rm -f $tempPlan
    rm -rf $tempDir
}

awsAccountId=$(getAwsAccountId)

# Test for incompatible options
# -t and -d (plan and destroy called)

terraformInit
[[ $setWorkspace != "false" ]] && terraformCreateWorkspace
[[ $destroy == "false" ]] && terraformPlan
[[ $planOnly != "true" && $destroy == "false" ]] && terraformApply
[[ $planOnly != "true" && $destroy == "true" ]] && terraformDestroy
[[ $cleanUp != "false" ]] && cleanUp
