envvars := "config.env"

set dotenv-filename := "config.env"

container_cmd := env_var_or_default("COMPOSE_CMD", "docker-compose")
# This is the production bowtienet/registry bowtie-server image.
#
# To use the staging/release candidate image, use 5633314 instead.
container_registry_id := env_var_or_default("REGISTRY_ID", "5654678")

# Print usage
help:
	@just --list

# Generate user documentation
generate:
	go generate ./...

# Ensure documentation is up-to-date
stale-docs: generate
	#!/usr/bin/env bash

	if git diff --no-ext-diff --quiet --exit-code docs
	then
		echo "Documentation is up-to-date"
	else
		echo -e "\n[ ! ] Documentation is out-of-date with source.\n"
		echo "Regenerate and commit updated docs with 'just generate'."
		exit 1
	fi

# Perform documentation checks
stylecheck: generate
	vale docs

# Run the tests
test:
	go test -v ./... -count=1

# Run all tests, including acceptance tests
acceptance-test: container
	#!/usr/bin/env bash
	# Ensure the container has had time to come up
	sleep 5
	# Run the tests
	TF_ACC=1 just test
	# Save the return code, we want to ensure that we shut down the
	# container before completing
	result=$?
	# Shut down the container
	just stop-container || true
	# Exit code of the actual tests:
	exit $result

# Generate a SITE_ID for the test container in config.env
site-id:
	#!/usr/bin/env bash

	conf={{envvars}}
	if grep SITE_ID $conf &>/dev/null
	then
		echo "SITE_ID present in $conf"
	else
		set -x
		echo "SITE_ID=$(uuidgen)" >> $conf
	fi

# Ensure that the BOWTIE_HOST env var is set
bowtie-host:
	#!/usr/bin/env bash

	conf={{envvars}}
	var=BOWTIE_HOST
	if grep $var $conf &>/dev/null
	then
		echo "$var present in $conf"
	else
		port=$(yq -r '.services.bowtie.ports[0] | split(":")[0]' < compose.yaml)
		set -x
		echo "$var=http://127.0.0.1:$port" >> $conf
	fi

# Generate an init-users file for bootstrapping
init-users:
	#!/usr/bin/env bash

	users_file=container/init-users

	if [[ -e $users_file ]]
	then
		echo "$users_file  exists; use 'just clean' to purge container state"
		exit
	fi

	sed -i '/BOWTIE_USERNAME/d;/BOWTIE_PASSWORD/d' {{envvars}}
	username=admin@example.com
	password=$(openssl rand -hex 16)
	hash=$(echo -n $password | argon2 $(uuidgen) -i -t 3 -p 1 -m 12 -e)
	echo $username:$hash > $users_file
	echo "BOWTIE_PASSWORD=$password" >> {{envvars}}
	echo "BOWTIE_USERNAME=$username" >> {{envvars}}
	echo "Generated user $username"

# Pull the latest image tag and set it as an environment variable
image-var registry=container_registry_id:
    #!/usr/bin/env python3

    from concurrent.futures import ThreadPoolExecutor
    from datetime import datetime
    from pathlib import Path
    from pprint import pprint

    import requests
    import yaml

    page = 1
    url = "https://gitlab.com/api/v4/projects/bowtienet%2Fregistry/registry/repositories/{{registry}}/tags"
    tags = []

    while page:
        response = requests.get(url, params={"page": page})
        tags.extend(response.json())
        page = response.headers.get("x-next-page")

    with ThreadPoolExecutor() as executor:
        all_tags = executor.map(
            lambda t: requests.get(f"{url}/{t['name']}").json(),
            tags
        )

    latest = sorted(
        all_tags,
        key=lambda t: datetime.fromisoformat(t['created_at'])
    )[-1]

    print(f"Setting image to {latest['location']}")
    compose = Path("compose.yaml")
    with compose.open('r') as handle:
        y = yaml.load(handle, Loader=yaml.CLoader)
    y['services']['bowtie']['image'] = latest['location']
    with compose.open('w') as handle:
        yaml.dump(y, handle, Dumper=yaml.CDumper)

# Start a background container for bowtie-server
container cmd=container_cmd: site-id init-users image-var bowtie-host
	{{cmd}} up --detach

# Stop the background container
stop-container cmd=container_cmd:
	{{cmd}} down

# Remove build and container artifacts
clean:
	git clean -f -d -x container/
	cat /dev/null > {{envvars}}

sweep:
	go test ./internal/bowtie/test -v -sweep=http://localhost:3000
