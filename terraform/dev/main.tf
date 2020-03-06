provider "google" {
  version = "~> 1.20"
  project = "newscorp-newsiq-dev"
  region  = "us-east1"
}

provider "google-beta" {
  version = "~> 2.11.0"
  project = "newscorp-newsiq-dev"
  region  = "us-east1"
}

provider "kubernetes" {
  version = "~> 1.5"
}

provider "null" {
  version = "~> 2.0"
}

provider "template" {
  version = "~> 2.0"
}

module "prebid" {
  source            = "../tfmodules/prebid"
  gcp_projectid     = "newscorp-newsiq-dev"
  gcp_projectnumber = 16509401209
  environment       = "dev"

  # BigTable not available in us-east4
  region                              = "us-east1"
  primary_zone                        = "us-east1-c"
  loadbalancer_tls_private_key        = "${file("secrets-repo/others/server.key")}"
  loadbalancer_tls_certificate        = "${file("secrets-repo/others/server.crt")}"
  bigtable_num_nodes                  = 1
}

module "gke" {
  enabled           = 1
  source            = "../tfmodules/gke"
  gcp_projectid     = "newscorp-newsiq-dev"
  gcp_projectnumber = 305210329360
  region            = "us-east1"
  primary_zone      = "us-east1-c"
  autoscaling_min   = 1
  autoscaling_max   = 6
  vm_type           = "n1-standard-2"
}
