resource "superset_group_role_binding" "example" {
  group_name = "Group1"
  role_names = [
    "Public"
  ]
}
