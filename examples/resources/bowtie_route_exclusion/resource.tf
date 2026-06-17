# Keep an internal CIDR collection out of the Bowtie tunnel on every site.
resource "bowtie_route_exclusion" "printers" {
  name          = "Office printers"
  collection_id = bowtie_collection.printers.id
}

# Scope an exclusion to specific sites and the office WAN, for laptops only.
resource "bowtie_route_exclusion" "office_lan" {
  name          = "Office LAN"
  collection_id = bowtie_collection.office_lan.id
  sites         = [bowtie_site.hq.id]

  only_if_wan_matches_cidrs = ["203.0.113.0/24"]
  match_only_device_type    = "laptop"
}
