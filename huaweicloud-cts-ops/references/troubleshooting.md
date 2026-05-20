# Huawei Cloud CTS Troubleshooting

## Common Failure Scenarios

### 1. Trail creation fails

- Symptoms:
  - `Cts.0401` duplicate trail name
  - `Cts.0402` invalid destination config
  - `Cts.0403` region unsupported
  - `Cts.0405` quota exceeded

- Action:
  - Confirm the trail name is unique.
  - Validate the delivery target configuration.
  - Confirm the target region supports CTS.
  - Check project trail quotas and delete unused trails.

### 2. Query returns zero results

- Symptoms:
  - No events found despite recent activity.
  - Query executes with success but empty result set.

- Action:
  - Confirm the query time range covers event generation time.
  - Relax or simplify the filter expression.
  - Check that the trail is active and correctly configured.
  - Validate the event source service is supported by CTS.

### 3. Delivery destination not receiving events

- Symptoms:
  - Trail status is `ACTIVE`, but OBS/SMN/LTS has no new files or messages.
  - Delivery failures appear in CTS console.

- Action:
  - Validate the destination exists and the target config is correct.
  - Confirm IAM policies allow CTS to write to OBS/SMN/LTS.
  - Check destination permissions and bucket/sink access.
  - Verify service endpoint connectivity and region alignment.

### 4. Authorization failures

- Symptoms:
  - CLI/SDK returns `Unauthorized`, `AccessDenied`, or `InvalidCredentials`.

- Action:
  - Ensure `HW_ACCESS_KEY_ID` and `HW_SECRET_ACCESS_KEY` are valid.
  - Confirm the access key has CTS read/write permissions.
  - Verify the project scope via `HW_PROJECT_ID`.
  - If using temporary credentials, verify token validity.

### 5. Throttling and rate limits

- Symptoms:
  - `429 Too Many Requests` or internal rate-limit errors.

- Action:
  - Retry with exponential backoff.
  - Reduce request frequency.
  - Aggregate queries when possible.
  - Contact Huawei Cloud if rate limits are consistently exceeded.

## Diagnostic Checklist

1. Validate CLI support:

    ```bash
    hcloud cts list-trails --region {{env.HW_REGION_ID}}
    ```

2. Validate SDK call:

    ```go
    response, err := client.ListTrails(context.TODO(), &model.ListTrailsRequest{})
    ```

3. Check trail status:

    - `ACTIVE`: normal
    - `CREATING`: wait
    - `FAILED`: inspect destination config
    - `DELETING`: confirm deletion in progress

4. Check delivery target health:

    - OBS buckets: read/write permissions
    - SMN topics: publish permissions
    - LTS log groups: sink configuration

5. Review query filter syntax and time range.

## Recovery Patterns

- If a trail is stuck in `CREATING`, verify destination connectivity and retry after a short delay.
- If a trail fails repeatedly, delete and recreate it with a simpler delivery configuration.
- If event delivery is missing, temporarily switch to a different supported target such as OBS.
- If queries are inconsistent, compare time ranges across multiple trails.

## Alertworthy Conditions

- Audit trail deletion without explicit confirmation.
- Persistent audit delivery failures.
- Repeated query failures for valid time windows.
- Unauthorized CTA API calls from administrative accounts.
