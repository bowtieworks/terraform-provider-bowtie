# [Terraform Provider](https://registry.terraform.io/providers/bowtieworks/bowtie/latest)  for [Bowtie](https://docs.bowtie.works)

This provider enables configuration for Bowtie deployments via Terraform.

## Using this provider

In our example we'll set up these variables:

    export TF_VAR_bowtie_admin_user=issac@bowtie.works
    export TF_VAR_bowtie_admin_password=hunter2
    export TF_VAR_bowtie_host=https://canary-8.net.rock.associates.example

And then configuring a few dns records would look something like this:

    terraform {
      required_providers {
        bowtie = {
            source = "bowtieworks/bowtie"
            version = "0.6.0"
        }
      }
    }

    variable bowtie_host {
        type = string
        nullable = false
    }

    variable bowtie_admin_user {
        type = string
        nullable = false
    }

    variable bowtie_admin_password {
        type = string
        nullable = false
    }


    provider "bowtie" {
        host     = var.bowtie_host
        username = var.bowtie_admin_user
        password = var.bowtie_admin_password
    }

    resource "bowtie_dns" "freshbooks" {
        name = "rock.associates.com"
        servers = [{
            addr = "172.128.40.78",
        },
        {
            addr = "172.16.40.199",
        }]
        is_dns64 = true
        is_drop_all = false
        is_drop_a = true
        is_log = true
        excludes = []
    }

    resource "bowtie_dns" "contoso" {
        name = "contoso.com"
        servers = [{
            addr = "9.9.9.9",
        },
        {
            addr = "1.1.1.1",
        }]
        is_drop_all = true
        is_drop_a = true
        is_log = true
        excludes = []
    }

Then you can run `terraform plan` and `terraform apply` as usual

## Development

### Building

Setup your dev environment:

    terraform-provider-bowtie on  main [!?] via 🐹 v1.21.2 via 🐍 v3.10.12 (env) 
    ❯ cat ~/.terraformrc 
    provider_installation {
      dev_overrides {
        "registry.terraform.io/bowtieworks/bowtie" = "/Users/Username/bowtie/terraform-provider-bowtie"
      }
    }

go build -o terraform-provider-bowtie

### Generate Docs

`just generate`

### Testing

In addition to basic unit tests that can be run with `go test ./...` from the root of the repository, acceptance tests are also available.
There is a small degree of setup to run acceptance tests because they require a local Bowtie API to perform requests against.

Enter the devshell (with either `direnv allow` in this directory, `nix develop .`, or `nix-shell` or, alternatively, install the development dependencies manually:

- `just` to drive tasks
- `argon2` for password hash generation
- `httpie` for container health checks
- `go`

Additionally, you will need a functional container runtime to operate the Bowtie server container.
A `compose.yml` file is provided and knowing working configurations include `docker`, `podman`, and `finch`.
Override the `COMPOSE_CMD` in `.envrc.local` if you’re using something other than `docker-compose`.

With the prerequisites satisfied, run the acceptance tests with:

	just acceptance-test

For a pristine environment afterward, you may `just clean` to remove leftover container files in `./container`.
