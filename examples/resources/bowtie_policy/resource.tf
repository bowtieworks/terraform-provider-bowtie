# Grant a user group access to a resource group.
resource "bowtie_policy" "engineering_access" {
  source = {
    user_group = bowtie_group.engineering.id
  }
  dest   = bowtie_resource_group.internal_tools.id
  action = "Accept"
}

# Deny everyone access to a sensitive resource group, evaluated first.
resource "bowtie_policy" "deny_finance" {
  source = {
    always = true
  }
  dest   = bowtie_resource_group.finance.id
  action = "Reject"
  order  = 0
}

# Compose matchers: accept if the device is in either device group.
resource "bowtie_policy" "managed_devices" {
  source = {
    or = [
      { device_group = bowtie_device_group.corporate_laptops.id },
      { device_group = bowtie_device_group.corporate_phones.id },
    ]
  }
  dest   = bowtie_resource_group.internal_tools.id
  action = "Accept"
}
