resource "superset_role_permissions" "example" {
  role_name = "Role1"
  permissions = [
    { permission_name = "can_post", view_menu_name = "Group" },
    { permission_name = "can_get", view_menu_name = "Group" },
  ]
}
