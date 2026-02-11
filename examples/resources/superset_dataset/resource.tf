resource "superset_dataset" "example" {
  table_name            = "table_name"
  schema                = "sample_schema"
  is_managed_externally = true
  database_name         = "PostgreSQL_DB"
}
