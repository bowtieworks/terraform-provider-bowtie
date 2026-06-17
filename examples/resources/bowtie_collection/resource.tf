resource "bowtie_collection" "internal_endpoints" {
  name        = "Internal Endpoints"
  description = "Private addresses and domains used by internal services."

  members = [
    {
      name     = "Wiki"
      location = { dns = "wiki.internal.example.com" }
    },
    {
      name     = "Datacenter range"
      comment  = "Primary east-coast subnet"
      location = { cidr = "10.10.0.0/16" }
    },
    {
      name     = "Temporary vendor host"
      expires  = "2026-12-31T23:59:59Z"
      location = { ip = "203.0.113.42" }
    },
  ]
}
