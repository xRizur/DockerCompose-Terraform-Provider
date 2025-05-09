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