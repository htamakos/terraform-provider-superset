


- get_slack_channels_schema の追加
    - https://github.com/apache/superset/blob/c31224c891d3269893d3724c62777ab6557d598d/superset/reports/schemas.py#L52
- ErrorMessageSchema の追加
    - 410 のネストされたスキーマを shcmeas: 以下に移動
- DashboardColorsConfigUpdateSchema の追加
    - https://github.com/apache/superset/blob/c31224c891d3269893d3724c62777ab6557d598d/superset/dashboards/schemas.py#L453
- DashboardNativeFiltersConfigUpdateSchema の追加
    - https://github.com/apache/superset/blob/c31224c891d3269893d3724c62777ab6557d598d/superset/dashboards/schemas.py#L447

- get_fav_star_ids_schema 名を get_fav_star_ids_only_schema にリネーム
