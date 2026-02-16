resource "superset_dataset_folder" "example" {
  dataset_name = superset_dataset.example.table_name
  folders = [
    {
      name = "test_folder"
      children = [
        {
          name        = "COL1"
          type        = "column"
          description = "COL1"
        },
        {
          name        = "COL2"
          type        = "column"
          description = "COL2"
        },
      ]
      type = "folder"
    },
    {
      name = "test_folder2"
      children = [
        {
          name        = "METRICS1"
          type        = "metric"
          description = "METRICS1"
        },
      ]
      type = "folder"
    }
  ]
}
