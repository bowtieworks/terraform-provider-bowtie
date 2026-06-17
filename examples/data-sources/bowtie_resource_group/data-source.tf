data "bowtie_resource_group" "internal_tools" {
  name = "Internal Tools"
}

# Reference the looked-up ID as a policy destination.
resource "bowtie_policy" "engineering_access" {
  source = {
    user_group = bowtie_group.engineering.id
  }
  dest   = data.bowtie_resource_group.internal_tools.id
  action = "Accept"
}
