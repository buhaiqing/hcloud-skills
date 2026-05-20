# Huawei Cloud CTS CLI Usage

## CLI Overview

Huawei Cloud CTS access is available via `hcloud cts` commands. The CLI path is useful for quick verification and one-off operations.

## Common CTS Commands

- `hcloud cts list-trails`
- `hcloud cts show-trail --trail-id <trail-id>`
- `hcloud cts create-trail --name <name> --delivery-to <OBS|SMN|LTS> --delivery-config <config>`
- `hcloud cts update-trail --trail-id <trail-id> --delivery-config <config>`
- `hcloud cts delete-trail --trail-id <trail-id>`
- `hcloud cts query-events --trail-id <trail-id> --start-time <start> --end-time <end> --filter <expr>`

## Typical Command Patterns

### List Trails

```bash
hcloud cts list-trails --region {{env.HW_REGION_ID}} --limit 100
```

### Show Trail Details

```bash
hcloud cts show-trail \
  --region {{env.HW_REGION_ID}} \
  --trail-id "{{output.trail_id}}"
```

### Create a Trail

```bash
hcloud cts create-trail \
  --region {{env.HW_REGION_ID}} \
  --name "{{user.trail_name}}" \
  --delivery-to "OBS" \
  --delivery-config '{"bucket_name":"my-cts-bucket","location":"cn-north-4"}' \
  --retention-days 365
```

### Query Events

```bash
hcloud cts query-events \
  --region {{env.HW_REGION_ID}} \
  --trail-id "{{output.trail_id}}" \
  --start-time "{{user.query_start_time}}" \
  --end-time "{{user.query_end_time}}" \
  --filter "{{user.query_filter}}" \
  --limit 100
```

## CLI Coverage Notes

| Feature | CLI Supported | Remarks |
|---------|---------------|---------|
| Create trail | Yes | Primary user path |
| Update trail | Yes | Use delivery-config patch |
| Delete trail | Yes | Confirm before deletion |
| Query events | Yes | Useful for fast forensic lookup |
| Show event detail | Partial | May require SDK for advanced query fields |
| Cross-region correlation | No | Trail scope is region-specific |

## CLI Troubleshooting

- If `hcloud cts` commands return `command not found`, verify CLI installation and `PATH`.
- If `Unauthorized` or `AccessDenied`, check `HW_ACCESS_KEY_ID` and `HW_SECRET_ACCESS_KEY`.
- If `InvalidTrailId`, verify the provided trail ID with `list-trails`.
- If query returns empty results, broaden time range or simplify filter expression.

## CLI Best Practices

- Use `--region` explicitly for all CTS commands.
- Store long JSON delivery configurations in a file and pass via `--delivery-config @config.json`.
- Prefer `hcloud cts` for operational validation and use SDK for automation and complex queries.
