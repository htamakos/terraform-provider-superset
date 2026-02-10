resource "superset_user" "example" {
  username   = "example_user"
  first_name = "FirstName"
  last_name  = "LastName"
  email      = "example@example.com"
  password   = "password"
  role_names = [
    "Gamma",
    "Alpha"
  ]
  group_names = [
    "Group1"
  ]
}
