terraform {
  required_providers {
    bowtie = {
      source  = "registry.terraform.io/bowtieworks/bowtie"
      version = "0.0.1"
    }
  }
}

# Set your username and password by exporting credentials to the
# BOWTIE_USERNAME and BOWTIE_PASSWORD environment variables.

provider "bowtie" {
  host = "https://bowtie.example.com"
}

# Set your username and password by exporting credentials to the
# TF_VAR_bowtie_username and TF_VAR_bowtie_password environment
# variables. Note that you must also define these variables in
# `variable bowtie_username { }` and `variable bowtie_password { }`
# blocks.

provider "bowtie" {
  host     = "https://bowtie.example.com"
  username = var.bowtie_username
  password = var.bowtie_password
}

# Set your username and password with plaintext values (not recommended)

provider "bowtie" {
  host     = "https://bowtie.example.com"
  username = "example"
  password = "test1123"
}
