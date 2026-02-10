# terraform-provider-superset

Terraform Provider for **Apache Superset**

This Terraform provider enables you to manage Apache Superset resources declaratively via Terraform configurations.  
It extends Terraform with Superset-specific resources and data sources so that dashboards, roles, users, and other Superset objects can be created, updated, and deleted through Terraform workflows.

---

## 🚀 Features

This provider integrates Terraform with the Apache Superset REST API and currently supports:

- Managing Superset roles
- Managing Superset users
- Managing Superset groups
- Managing Superset role permissions
- Managing Superset group role assignments
- (Add other supported resources here)

Resources can be imported into Terraform state where supported.

---

## 📦 Installation

### Requirements

- Terraform CLI >= 1.0
- Go >= 1.24 (for building the provider)
- An accessible Apache Superset instance

### Build from Source

```bash
git clone https://github.com/htamakos/terraform-provider-superset.git
cd terraform-provider-superset
go install
```

This installs the provider binary into your `$GOPATH/bin`.

---

## 🔌 Provider Configuration

Configure the provider in your Terraform project:

```hcl
terraform {
  required_providers {
    superset = {
      source  = "htamakos/superset"
      version = "0.1.0"
    }
  }
}

provider "superset" {
  host     = "https://superset.example.com"
  username = "admin"
  password = "super_secret"
}
```

### Provider Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | yes | Base URL of the Superset instance |
| `username` | string | yes | Superset login username |
| `password` | string | yes | Superset login password (sensitive) |

Environment variable authentication may also be supported depending on configuration.

---

## 📘 Example Usage

### Create a Superset Role

```hcl
resource "superset_role" "example" {
  name        = "terraform_role"
  permissions = [
    "can_read",
    "can_write"
  ]
}
```

### Import an Existing Role

```bash
terraform import superset_role.example 632
```

---

## 📚 Resources and Data Sources

### Resources

| Name | Description |
|------|-------------|
| `superset_role` | Manage Superset roles |
| `superset_user` | Manage Superset users |
| *(add additional resources here)* | |

### Data Sources

| Name | Description |
|------|-------------|
| `superset_roles` | Retrieve existing Superset roles |
| `superset_users` | Retrieve existing Superset users |

---

## 🧪 Development

### Local Development

```bash
go install
```

### Generate Documentation

```bash
make generate
```

### Run Acceptance Tests

```bash
make testacc
```

⚠️ Acceptance tests require a running Superset instance and will perform real API operations.

---

## 📄 License

This project is licensed under the **Mozilla Public License 2.0 (MPL-2.0)**.

---

## 🤝 Contributing

Issues and pull requests are welcome.  
Please ensure new resources and changes include appropriate documentation and tests.

