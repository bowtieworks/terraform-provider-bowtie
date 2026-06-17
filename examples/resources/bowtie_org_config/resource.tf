# The organization configuration is a singleton; import it first:
#   terraform import bowtie_org_config.this organization-config
resource "bowtie_org_config" "this" {
  # Organization-wide default update strategy, inherited by Controllers whose
  # own strategy is "org-default". This is fleet-level upgrade orchestration.
  controller_version_strategy_type  = "newest-at-calendar"
  controller_version_strategy_value = "*-*-* 03:00:00"
  controller_version_splay_type     = "consistent-randomized-delay"
  controller_version_splay_value    = "1h"
  controller_version_minimum_age    = 7

  # Zero-touch cluster scale-out.
  allow_controller_approval_with_psk_only = true

  # Surfaced fields are preserved when omitted. To remove a key from the
  # organization config document, list it here.
  clear_fields = [
    "controller_version_include_prereleases",
  ]
}
