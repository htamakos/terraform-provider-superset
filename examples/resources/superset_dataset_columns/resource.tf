resource "superset_dataset_columns" "example" {
  dataset_name = superset_dataset.example.table_name

  columns = {
    COL1 = {
      column_name  = "COL1"
      filterable   = true
      groupby      = true
      is_dttm      = false
      verbose_name = "COL1"
      is_active    = true
    },
    COL2 = {
      column_name  = "COL2"
      filterable   = true
      groupby      = true
      is_dttm      = true
      verbose_name = "COL2"
      is_active    = true
    },
  }
}
