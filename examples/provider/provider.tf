terraform {
  required_providers {
    sakura = {
      source = "htamakos/superset"

      version = "0.0.0"
    }
  }
}

# Configure the Superset Provider
# Note: The following configuration is for demonstration purposes only. In production, consider using environment variables or a secure secrets manager to handle sensitive information like credentials.
# For example, you can set the following environment variables:
# export SUPERSET_SERVER_BASE_URL="http://localhost:8080"
# export SUPERSET_USERNAME="username"
# export SUPERSET_PASSWORD="password"
provider "superset" {
  server_base_url = "http://localhost:8080"
  username        = "username"
  password        = "password"
}
