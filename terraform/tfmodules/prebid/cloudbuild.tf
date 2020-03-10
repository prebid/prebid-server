resource "google_cloudbuild_trigger" "keybase-builder" {

    description = "keybase-builder"

    trigger_template {
        project = "${var.gcp_projectid}"
        branch_name = "${var.environment == "prod" ? "master" : "develop"}"
        repo_name   = "github_newscorp-ghfb_prebid-server"
    }

    filename = "tools/keybase-cloudbuilder/cloudbuild.yaml"
}

resource "google_cloudbuild_trigger" "PrebidServer-Development" {

    description = "Push to Development Branch"

    trigger_template {
        project = "${var.gcp_projectid}"
        branch_name = "${var.environment == "prod" ? "master" : "develop"}"
        repo_name   = "github_newscorp-ghfb_prebid-server"
    }

    filename = "cloudbuild.yaml"

    substitutions {
        _CLOUDSDK_COMPUTE_ZONE = "us-east1-c"
        _CLOUDSDK_CONTAINER_CLUSTER = "kubernetes-prebid-cloudops"
        _ENV = "dev"
    }
}