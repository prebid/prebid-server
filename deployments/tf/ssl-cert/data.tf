# Find the environment-specific subdomain zone of raptive.com
data "aws_route53_zone" "env_adthrive_com" {
  name         = "${var.environment}.adthrive.com."
  private_zone = false
}
