#!/usr/bin/env bash
set -euo pipefail

# ------------------------------------------------------------
# tfplugindocs hermetic generator (HashiCorp-style headings)
# - Builds local provider
# - Captures providers schema via dev_overrides (no registry)
# - Duplicates schema key to include short name "bowtie"
# - Renders docs from templates/ into docs/
# - Uses short rendered name so page_title => "bowtie Provider"
# ------------------------------------------------------------

# ---- Settings
VERSION="${VERSION:-0.0.1}"
PROVIDER_SHORT="bowtie"
PROVIDER_FQN="registry.terraform.io/bowtieworks/bowtie"
ORG="bowtieworks"
NAME="${PROVIDER_SHORT}"

OS="$(go env GOOS)"
ARCH="$(go env GOARCH)"

# ---- Paths
ROOT="$PWD/.tf-plugins"
BUILD_DIR="$ROOT/dev/${ORG}/${NAME}"           # local binary lives here (dev_overrides points here)
HOME_DIR="$ROOT/home"                          # throwaway HOME for Terraform CLI config
SCHEMA_JSON="$ROOT/providers-schema.json"
SCHEMA_JSON_BOWTIE="$ROOT/providers-schema.bowtie.json"

mkdir -p "$BUILD_DIR" "$HOME_DIR"

# ---- Build local provider
BIN_PATH="$BUILD_DIR/terraform-provider-${NAME}_v${VERSION}"
GOFLAGS=${GOFLAGS:-}
GOOS="$OS" GOARCH="$ARCH" go build $GOFLAGS -o "$BIN_PATH"
chmod +x "$BIN_PATH"

# ---- Local Terraform CLI config: dev_overrides for BOTH addresses
TFRC="$HOME_DIR/.terraformrc"
cat > "$TFRC" <<RC
provider_installation {
  dev_overrides {
    "$PROVIDER_FQN"                        = "$BUILD_DIR"
    "registry.terraform.io/hashicorp/$NAME" = "$BUILD_DIR"
  }
  direct {}
}
RC

# Ensure Terraform uses our local CLI config/home (no registry calls for our provider)
export HOME="$HOME_DIR"
export TF_CLI_CONFIG_FILE="$TFRC"

# ---- Make a tiny temp module, init using dev_overrides (NO -plugin-dir), dump schema
tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

cat > "$tmpdir/main.tf" <<'HCL'
terraform {
  required_providers {
    bowtie = {
      source = "bowtieworks/bowtie"
      # no version; dev_overrides supplies local build
    }
  }
}
provider "bowtie" {}
HCL

# Terraform init must succeed WITHOUT hitting the registry for our provider
terraform -chdir="$tmpdir" init -input=false -no-color >/dev/null
terraform -chdir="$tmpdir" providers schema -json > "$SCHEMA_JSON"

# ---- Duplicate schema key so tfplugindocs can find either FQN or short name
jq '.provider_schemas["'"$PROVIDER_SHORT"'"] = .provider_schemas["'"$PROVIDER_FQN"'"]' \
  "$SCHEMA_JSON" > "$SCHEMA_JSON_BOWTIE"

# ---- Optional: format example snippets (doesn't fail the script if missing)
terraform fmt -recursive ./examples/ || true

# ---- Render docs
# NOTE: Using short rendered name ("bowtie") produces HashiCorp-style headings:
#   page_title: "bowtie Provider"
#   # bowtie Provider
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate \
  --providers-schema "$SCHEMA_JSON_BOWTIE" \
  --provider-name "$PROVIDER_SHORT" \
  --rendered-provider-name "$PROVIDER_SHORT" \
  --website-source-dir "templates" \
  --rendered-website-dir "docs"
