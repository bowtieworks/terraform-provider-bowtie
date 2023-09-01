---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "bowtie_user Data Source - terraform-provider-bowtie"
subcategory: ""
description: |-
  
---

# bowtie_user (Data Source)



## Example Usage

```terraform
data "bowtie_user" "admin" {
  email = "example@example.com"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `email` (String)

### Read-Only

- `authz_control_plane` (Boolean)
- `authz_devices` (Boolean)
- `authz_policies` (Boolean)
- `authz_users` (Boolean)
- `id` (String) The ID of this resource.
- `name` (String)
- `status` (String)