# Controllers register themselves with the control plane when they boot, so a
# Controller is imported and then managed, never created by Terraform:
#
#   terraform import bowtie_controller.east <controller-id>
#
# Once imported, manage its lifecycle settings as code.
resource "bowtie_controller" "east" {
  # Check for the newest release every night at 02:00, hold new releases back
  # for seven days, and spread restarts across the fleet so they do not all
  # update at once.
  version_strategy_type        = "newest-at-calendar"
  version_strategy_value       = "*-*-* 02:00:00"
  version_strategy_splay_type  = "consistent-randomized-delay"
  version_strategy_splay_value = "30m"
  version_minimum_age          = 7

  # Optional values that inherit organization defaults are preserved when
  # omitted. To clear a per-Controller override, list it here.
  clear_overrides = [
    "version_include_prereleases",
  ]
}
