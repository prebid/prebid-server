resource "google_compute_global_address" "kubernetes-ingress" {
  name       = "${var.override_ip_address_name}"
  ip_version = "IPV4"

  lifecycle {
    prevent_destroy = true
    ignore_changes  = "ip_version"
  }
}

data "google_compute_network" "default" {
  name = "default"
}

# The default-allow-internal rule allows all nodes to talk to all other nodes
# Without this rule, dataflow fails (hung / timeout on GRPC connections)
resource "google_compute_firewall" "default-allow-internal" {
  name        = "default-allow-internal"
  network     = "${data.google_compute_network.default.name}"
  description = "Allow internal traffic on the default network"
  direction   = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }

  source_ranges = [
    # The internal network CIDR
    "10.128.0.0/9",
  ]

  priority = 65534
}

# Disallow SSH
resource "google_compute_firewall" "deny-ssh" {
  name        = "deny-ssh"
  network     = "${data.google_compute_network.default.name}"
  description = "deny SSH"
  direction   = "INGRESS"

  deny {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = [
    "0.0.0.0/0",
  ]

  priority = 65534
}
