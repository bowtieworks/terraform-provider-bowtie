# Create a new IPv4 Pool
resource "bowtie_ipv4_pool" "device_pool" {
  range                      = "172.168.100.0/24"
  assign_addresses_from_here = "always-assign-random"
  skip_first_n_addresses     = 1
}
