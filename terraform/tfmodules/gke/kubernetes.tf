resource "google_container_cluster" "kubernetes-prebid-cloudops" {
  count                    = "${var.enabled ? 1 : 0}"
  name                     = "kubernetes-prebid-cloudops"
  zone                     = "${var.primary_zone}"
  initial_node_count       = 1
  remove_default_node_pool = true

  # Hardening
  enable_legacy_abac = false
  master_auth {
    username = ""
    password = ""

    client_certificate_config {
      issue_client_certificate = false
    }
  }

  # Restrict Master Node Access to our VPNs IP
  master_authorized_networks_config {

  }

  addons_config = {
    http_load_balancing {
      disabled = false
    }

    kubernetes_dashboard {
      disabled = true
    }
  }
  
  subnetwork = "default"
  min_master_version = "1.15.9-gke.8"
  node_version       = "1.15.9-gke.8"
  node_config {
    tags = ["gke-kubernetes-prebid-cloudops"]

    oauth_scopes = [
      "compute-rw",
      "storage-ro",
      "logging-write",
      "monitoring",
    ]
  }
}

#If we want to add nodes in a different zone
resource "google_container_node_pool" "prebid-node-pool" {
  count   = "${var.enabled ? 1 : 0}"
  name    = "prebid-node-pool"
  zone    = "${var.primary_zone}"
  cluster = "${google_container_cluster.kubernetes-newsiq.name}"

  management {
    auto_upgrade = true
  }

  autoscaling {
    min_node_count = "${var.autoscaling_min}"
    max_node_count = "${var.autoscaling_max}"
  }

  node_config {
    machine_type = "${var.vm_type}"

    oauth_scopes = [
      "compute-rw",
      "storage-ro",
      "logging-write",
      "monitoring",
    ]

    tags = ["gke-kubernetes-prebid-cloudops"]
  }
}
