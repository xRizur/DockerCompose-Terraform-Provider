# Terraform Provider for Docker Compose

This is a custom Terraform provider that allows you to manage Docker Compose stacks directly from your Terraform configuration. The provider enables you to define multi-container Docker applications using Terraform resources, and automatically generates and applies the corresponding `docker-compose.yml` file.

## Features

- **Manage Docker Compose stacks**: Define services, networks, and volumes as Terraform resources.
- **Automatic YAML generation**: The provider generates a valid `docker-compose.yml` file based on your Terraform configuration.
- **Lifecycle management**: Supports `create`, `read`, `update`, and `delete` operations for Docker Compose stacks.
- **Service configuration**: Configure images, ports, environment variables, commands, entrypoints, healthchecks, and more.
- **Network and volume support**: Define custom networks and volumes for your stack.

## How It Works

1. **Terraform Configuration**: You define your stack using the `dockercompose_stack` resource in your `.tf` files. Each service, network, and volume is described using nested blocks.

2. **Resource Logic**:
   - On `create` or `update`, the provider:
     - Parses the Terraform resource data.
     - Generates a `docker-compose.yml` file using Go templates.
     - Runs `docker compose up -d` to start the stack.
   - On `read`, the provider:
     - Checks if the `docker-compose.yml` file exists.
     - If missing, regenerates it from the Terraform state.
     - Runs `docker compose ps --services` to verify running containers and updates the Terraform state accordingly.
   - On `delete`, the provider:
     - Runs `docker compose down` to stop and remove the stack.

3. **State Management**: The provider uses the stack name as the resource ID and keeps the Terraform state in sync with the actual Docker Compose deployment.

## Example Usage

```hcl
terraform {
  required_providers {
    dockercompose = {
      source  = "local/dockercompose"
      version = "1.0.0"
    }
  }
}

provider "dockercompose" {}

resource "dockercompose_stack" "test" {
  name = "testapp"

  service {
    name  = "web"
    image = "nginx:latest"
    restart = "always"
    depends_on = ["db"]

    environment = {
      APP_ENV = "production"
      DEBUG   = "false"
    }

    command    = ["nginx", "-g", "daemon off;"]
    entrypoint = ["/docker-entrypoint.sh"]

    replicas = 3
  }

  service {
    name  = "db"
    image = "postgres:15"
    restart = "always"
    ports = ["5432:5432"]
    environment = {
      POSTGRES_USER     = "admin"
      POSTGRES_PASSWORD = "supersecret"
    }
  }

  network {
    name   = "backend-network"
    driver = "bridge"
  }
}
```

## Implementation Details

- The provider is implemented in Go and uses the [Terraform Plugin SDK v2](https://github.com/hashicorp/terraform-plugin-sdk).
- It uses Go templates and the [Sprig](https://github.com/Masterminds/sprig) library to generate the `docker-compose.yml`.
- The provider executes Docker Compose commands (`up`, `down`, `ps`) via the local shell.
- Helper functions are used to safely extract and convert values from the Terraform schema.

## Limitations

- The provider requires Docker and Docker Compose to be installed and available in the system's PATH.
- It currently manages the stack in the working directory and overwrites `docker-compose.yml`.
- Only a subset of Docker Compose features are supported; advanced configuration may require extending the provider.

