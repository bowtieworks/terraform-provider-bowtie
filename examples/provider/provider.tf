# Authenticate Terraform with a DEDICATED Bowtie service account: a local
# administrator created solely for automation and scoped to the least
# privilege it needs. Do not reuse a human administrator's credentials, and in
# an SSO-backed organization keep this as a separate local account so that
# automation never depends on a person's identity.
#
# Supply the account's credentials from a secrets manager or CI secret store
# through the BOWTIE_USERNAME and BOWTIE_PASSWORD environment variables (and
# BOWTIE_HOST for the endpoint). Leave them out of your Terraform configuration.

provider "bowtie" {
  host = "https://bowtie.example.com"
}

# Equivalent: inject the same credentials as Terraform variables (for example,
# sourced from HashiCorp Vault or a CI secret) via TF_VAR_bowtie_username and
# TF_VAR_bowtie_password. Declare matching `variable "bowtie_username" {}` and
# `variable "bowtie_password" {}` blocks marked `sensitive = true`.

provider "bowtie" {
  host     = "https://bowtie.example.com"
  username = var.bowtie_username
  password = var.bowtie_password
}

# A development controller with a self-signed certificate.

provider "bowtie" {
  host     = "https://bt0.dev.example.com"
  insecure = true
}

# A controller whose certificate is issued by a private certificate authority.

provider "bowtie" {
  host      = "https://bowtie.internal.example.com"
  ca_bundle = file("/etc/ssl/certs/internal-ca.pem")
}
