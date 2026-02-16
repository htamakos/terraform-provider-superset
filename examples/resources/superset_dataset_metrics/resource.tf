resource "superset_dataset_metrics" "example" {
  dataset_name = superset_dataset.example.table_name

  metrics = {
    METRIC1 = {
      metric_name  = "METRIC1"
      expression   = "SUM(COL1)"
      verbose_name = "METRIC1"
    },
  }
}
