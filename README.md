# [Terraform Provider](https://registry.terraform.io/providers/bowtieworks/bowtie/latest) for [Bowtie](https://docs.bowtie.works)

Manage a Bowtie deployment with Terraform instead of the Controller web
interface: the policy engine (policies, resources, resource groups,
collections, user and device groups), DNS, and organization settings.

## Using this provider

Configure the provider with your Controller endpoint and the credentials of a
Bowtie **service account**: a dedicated automation login, not a human
administrator. Supply the account's username and password through the
`BOWTIE_USERNAME` and `BOWTIE_PASSWORD` environment variables, sourced from a
secrets manager or CI secret store; the host can be set inline.

```hcl
terraform {
  required_providers {
    bowtie = {
      source = "bowtieworks/bowtie"
    }
  }
}

provider "bowtie" {
  host = "https://bowtie.example.com"
  # export BOWTIE_USERNAME=... and BOWTIE_PASSWORD=... before running Terraform
}
```

**Use a dedicated service account.** Create a local administrator in the
Controller that is used only by Terraform and grant it the least privilege it
needs. In an SSO organization keep it as a separate local account, so that
automation never depends on a person's identity and the account can be rotated
or disabled on its own. Keep its credentials in a secrets manager or CI secret
store and pass them through environment variables. Never commit them to your
Terraform configuration.

To reach a Controller that isn't fronted by a publicly trusted certificate, set
`insecure = true` (development Controllers with self-signed certificates) or
`ca_bundle` to a private CA's PEM (inline or a file path). Both also honor the
`BOWTIE_INSECURE` / `BOWTIE_CA_BUNDLE` environment variables.

A small slice of the policy engine, granting a group of devices access to an
internal resource:

```hcl
# A private resource and the group a policy can target.
resource "bowtie_resource" "wiki" {
  name     = "Internal Wiki"
  protocol = "https"
  location = { dns = "wiki.internal.example.com" }
  ports    = { range = [443, 443] }
}

resource "bowtie_resource_group" "internal_tools" {
  name      = "Internal Tools"
  inherited = []
  resources = [bowtie_resource.wiki.id]
}

# A device group used as the policy source.
resource "bowtie_device_group" "corporate_laptops" {
  name = "Corporate Laptops"
}

# Allow corporate laptops to reach the internal tools.
resource "bowtie_policy" "laptops_to_tools" {
  source = {
    device_group = bowtie_device_group.corporate_laptops.id
  }
  dest   = bowtie_resource_group.internal_tools.id
  action = "Accept"
}
```

Policy sources also compose with `and`, `or`, and `nor`. For example, accept if
the device is in either of two groups:

```hcl
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
```

Then run `terraform plan` and `terraform apply` as usual.

## Resources and data sources

Full reference documentation, including every attribute and import syntax, lives
in [`docs/`](./docs) and on the Terraform Registry.

**Policy engine**

- `bowtie_policy`: a rule granting or denying a source access to a resource
  group, with `always` / `authenticated_user` / `user` / `device` /
  `user_group` / `device_group` matchers and `and` / `or` / `nor` composition.
- `bowtie_resource` / `bowtie_resource_group`: network objects and the groups
  policies target.
- `bowtie_collection`: reusable sets of IPs, CIDRs, DNS names, or nested
  collections, with optional per-member expiry.
- `bowtie_group` / `bowtie_group_membership`: user groups and their members.
- `bowtie_device_group`: device groups referenced by policy sources.

**Access and routing**

- `bowtie_route_exclusion`: split-tunnel rules keeping a collection of CIDRs
  out of the tunnel, optionally scoped to sites, WAN networks, and devices.

**Controller lifecycle** (fleet management and upgrade orchestration as code):

- `bowtie_controller`: a Controller's update strategy, stagger, minimum
  release age, and pre-release opt-in (import-only; Controllers self-register).
- `bowtie_org_config`: organization-wide defaults, including the update
  strategy Controllers inherit, device-lifecycle automation, and quorum
  settings (singleton; import-only).
- `bowtie_ipv4_range` / `bowtie_ipv6_range`: organization address pools.

`bowtie_controller` and `bowtie_org_config` manage existing objects: import
them first, and note that destroy only stops Terraform from managing them. For
optional inherited/defaulted settings, omitting a field preserves the current
server value. Use `clear_overrides` on `bowtie_controller` or
`clear_fields` on `bowtie_org_config` when Terraform should actively remove a
previous override/default key. Destroying IPv4 or IPv6 range resources uses the
Controller's cascade delete behavior so existing allocations do not block
deletion.

**Network and identity**

- `bowtie_dns` / `bowtie_dns_block_list`: managed DNS and block lists.
- `bowtie_site` / `bowtie_site_range`: sites and their advertised ranges.
- `bowtie_user`, `bowtie_organization`.

**Data sources** look objects up by name so you can reference their IDs:
`bowtie_resource_group`, `bowtie_group`, `bowtie_device_group`,
`bowtie_collection`, `bowtie_user`, `bowtie_device`.

## Pulumi

The same resources are available through Pulumi. A
[`pulumi-terraform-bridge`](https://github.com/pulumi/pulumi-terraform-bridge)
wrapper generates a Pulumi provider from this Terraform provider, so every
Bowtie resource and data source is usable from Pulumi in TypeScript, Python, Go,
and .NET. The bridge and its generated SDKs live in [`pulumi/`](./pulumi). See
[`pulumi/README.md`](./pulumi/README.md) for how it works and how to regenerate
it after a schema change.

## Development

### Building

Add a development override to `~/.terraformrc` so Terraform uses your local
build instead of a published release:

```hcl
provider_installation {
  dev_overrides {
    "bowtie.works/bowtie/bowtie" = "/path/to/terraform-provider-bowtie"
  }
  direct {}
}
```

Then build the binary:

```sh
go build -o terraform-provider-bowtie
```

Regenerate the documentation and format examples after changing a schema:

```sh
go generate ./...
```

### Testing

Unit tests run without any external dependencies:

```sh
go test ./...
```

Acceptance tests exercise the provider against a real Bowtie API and require a
little more setup. Enter the devshell (`direnv allow`, `nix develop .`, or
`nix-shell`) or install the dependencies manually:

- `just` to drive tasks
- `argon2` for password hash generation
- `httpie` for container health checks
- `go`

A functional container runtime is also required to run the Bowtie server
container; `docker`, `podman`, and `finch` are all known to work. A
`compose.yaml` file is provided. Override `COMPOSE_CMD` in `.envrc.local` if
you use something other than `docker-compose`.

With the prerequisites satisfied, run:

```sh
just acceptance-test
```

For a pristine environment afterward, `just clean` removes leftover container
files in `./container`.
