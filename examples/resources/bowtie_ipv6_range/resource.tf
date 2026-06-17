# Destroying this resource uses the Controller's cascade delete behavior so
# existing allocations from the pool do not block deletion.
resource "bowtie_ipv6_range" "lan" {
  # Leave range unset to let the Controller generate a Bowtie ULA prefix.
  assign_addresses_from_here = true
}
