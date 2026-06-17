# Destroying this resource uses the Controller's cascade delete behavior so
# existing allocations from the pool do not block deletion.
resource "bowtie_ipv4_range" "lan" {
  range                      = "192.0.2.0/24"
  assign_addresses_from_here = "always-assign-random"
  skip_first_n_addresses     = 1
}
