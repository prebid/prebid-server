output "ssl-cert-west-arn" {
  value = aws_acm_certificate.cert.arn
}

output "ssl-cert-east-arn" {
  value = aws_acm_certificate.cert_use1.arn
}

output "ssl-cert-experiment-west-arn" {
  value = aws_acm_certificate.cert_experiment.arn
}

output "ssl-cert-experiment-east-arn" {
  value = aws_acm_certificate.cert_experiment_use1.arn
}

output "ssl-cert-main-east-arn" {
  value = aws_acm_certificate.main_cert_use1.arn
}

output "ssl-cert-main-west-arn" {
  value = aws_acm_certificate.cert-west.arn
}