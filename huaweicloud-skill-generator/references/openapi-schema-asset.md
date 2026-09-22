# OpenAPI Schema Asset (hallucination L2)

> **Purpose**: A generated skill MUST ship `references/openapi-schema.json` whenever
> an upstream OpenAPI (or SDK model) contract is available for that product. The GCL
> `[H]` stage validates every Generator output document against this file **before**
> the Critic runs — a skill without it silently loses one of its three hallucination
> layers.
> **Consumer**: `hwcloud-skillcheck/internal/gcl/hallucination_l2.go`
> (`DefaultHallucinationDetector.checkJSONStructure`) via `internal/schema.ValidateFile`.
> **Version**: 1.0.0
> **Last Updated**: 2026-09-20

---

## 1. Placement and role

| Item | Value |
|------|-------|
| File path | `<skill-dir>/references/openapi-schema.json` (exact name and location — the loader joins the skill root with `references/openapi-schema.json`) |
| Schema root | JSON object; it MUST resolve to an object, else the check reports `schema root must resolve to an object` |
| Instance validated | The Generator's `ResultExcerpt` — the raw JSON document a CLI/SDK command returned, not the prompt and not the skill's own assets |
| On violation | L2 blocks the run: `hallucination_detection.l2.blocked = true` → `status: SAFETY_FAIL`, `hallucination_blocked: true`, `failure_pattern.category: hallucination` |
| On absence | No check runs; the runner emits one `slog.Warn` per skill and L2 reports `skipped_no_schema` |

```text
references/openapi-schema.json ─┐
                                ├─→ schema.ValidateFile(ResultExcerpt, schema) ─→ L2Result
Generator command stdout (JSON) ┘
```

The check is a **blocking** gate, so a schema that over-claims fields turns correct
CLI output into `SAFETY_FAIL`. Assert only what the upstream contract states.

## 2. Generation step (add to the generator flow)

1. Collect the operations the skill actually calls — the operation map written to
   `references/api-sdk-usage.md` — plus the response body each one returns.
2. Take those response schemas from the product's OpenAPI document (or the Go SDK
   response models when no OpenAPI document is published) and assemble them into one
   schema document using the shape in §3.
3. Reduce every node to the keywords in §4. Delete keywords the validator ignores
   rather than leaving them in place: an ignored `oneOf` reads like a real constraint
   to a human reviewer but asserts nothing.
4. Inline each `$ref`, or hoist it into a root-level `$defs` map (only
   `#/$defs/<name>` is resolvable — §5).
5. Verify before committing: the file parses as JSON, uses only §4 keywords, and every
   `$ref` names an existing `$defs` entry.

```bash
python3 - <<'PY'
import json, pathlib

NAME_MAPS = {"$defs", "properties"}  # maps of name → subschema, not keywords
ALLOWED = {"$defs", "$ref", "type", "const", "enum", "minimum", "maximum",
           "format", "minLength", "required", "properties",
           "additionalProperties", "items", "minItems"}

doc = json.loads(pathlib.Path("huaweicloud-<product>-ops/references/openapi-schema.json").read_text())
defs = set(doc.get("$defs", {}) or {})
seen, refs = set(), []

def walk(node):
    if isinstance(node, dict):
        for k, v in node.items():
            if k in NAME_MAPS:
                for sub in (v or {}).values():
                    walk(sub)
                continue
            seen.add(k)
            if k == "$ref" and isinstance(v, str):
                refs.append(v)
            walk(v)
    elif isinstance(node, list):
        for v in node:
            walk(v)

walk(doc)
unknown = sorted(seen - ALLOWED)
bad_refs = [r for r in refs if not r.startswith("#/$defs/") or r.rsplit("/", 1)[-1] not in defs]
print("unsupported keywords:", unknown or "none")
print("unresolvable refs:", bad_refs or "none")
assert not unknown and not bad_refs, "fix the schema before committing"
PY
```

## 3. Document shape

Recommended for response documents (single contract, reusable `$defs`):

```json
{
  "$defs": {
    "server": {
      "type": "object",
      "required": ["server_id", "status"],
      "properties": {
        "server_id": { "type": "string" },
        "status": { "type": "string", "enum": ["ACTIVE", "SHUTOFF", "BUILD", "ERROR"] },
        "created_at": { "type": "string", "format": "date-time" },
        "addresses": { "type": "object" }
      },
      "additionalProperties": true
    }
  },
  "$ref": "#/$defs/server"
}
```

A plain schema without `$defs` is equally valid when the skill's responses are a single
flat document:

```json
{
  "type": "object",
  "required": ["server_id"],
  "properties": { "server_id": { "type": "string" } }
}
```

Both shapes are exercised by `hwcloud-skillcheck/internal/gcl/hallucination_l2_test.go`
(`l2ValidSchema`, `l2DefsSchema`).

## 4. Keyword contract (exhaustive — the validator implements nothing else)

| Keyword | Accepted form | Behaviour |
|---------|---------------|-----------|
| `type` | string or array of strings | One of `null`, `boolean`, `integer`, `number`, `string`, `array`, `object`. A mismatch ends validation of that node. An `integer` satisfies `number`; JSON numbers containing `.`, `e` or `E` are `number`, others are `integer` |
| `enum` | array | JSON-equality membership; violation message `expected one of [...]` |
| `const` | any JSON value | JSON-equality equality |
| `minimum` / `maximum` | number | Numeric bounds only |
| `format` | `"date-time"` | RFC3339 check; any other format string is ignored |
| `minLength` | number | String length only |
| `required` | array of property names | Missing property → `missing required property "x"` |
| `properties` | map of name → subschema | Applied only to keys that are present |
| `additionalProperties` | `false` or a subschema | `false` rejects unknown keys; a subschema validates them |
| `items` | subschema | Applied to every array element |
| `minItems` | number | Array length floor |

Not implemented (silently ignored — do not rely on them): `oneOf`, `anyOf`, `allOf`,
`not`, `pattern`, `patternProperties`, `propertyNames`, `maxLength`, `maxItems`,
`uniqueItems`, `exclusiveMinimum` / `exclusiveMaximum`, `multipleOf`, `minProperties`,
`maxProperties`, `default`, `examples`, `deprecated`, `format` values other than
`date-time`, and all OpenAPI-only keys (`openapi`, `paths`, `components`).

## 5. `$ref` rules

- Only local `#/$defs/<name>` references are resolvable. `$defs` MUST be a root-level object.
- A `$ref` naming a missing entry is a loud failure: `unknown schema $ref: ...` →
  L2 status `validator_error`.
- Any other ref form (`#/definitions/x`, `#/components/schemas/x`, `other.json#/x`) is
  **not** resolved. The `$ref` key is dropped and the remaining keys of that node are
  kept — the node then asserts nothing at all. Inline those refs instead.
- Do not ship a full OpenAPI document (`paths` / `components`) as this asset: it is
  parsed as a schema, so the response shapes would never be reached.

## 6. Outcome statuses (the honesty contract)

`(*L2Result).Status()` reports how the check concluded, so downstream readers can tell
a skip from a pass:

| Status | Meaning | Blocks |
|--------|---------|--------|
| `pass` | Schema present and the output conformed | no |
| `violation` | Output violated the schema; errors recorded in `l2.errors` | yes |
| `skipped_no_schema` | This skill ships no `references/openapi-schema.json` — **not** a pass | no (one `slog.Warn` per skill, per process) |
| `skipped_no_output` | Generator produced no result excerpt to validate | no |
| `invalid_output` | Result excerpt is not JSON, so nothing could be compared | no |
| `schema_unreadable` | The asset exists but could not be read | no |
| `validator_error` | The asset is not valid JSON, or a `$ref`/root cannot be resolved | no |

The L2 block is persisted under `hallucination_detection` in the GCL trace when a
violation blocks the run; skipped/pass statuses are surfaced through the status method
and the skip warning.

## 7. Anti-patterns

- **Placeholder schemas.** `{"type":"object"}` validates everything, so L2 passes while
  checking nothing. If no contract is available, ship no file and let L2 report
  `skipped_no_schema` — a visible capability gap beats a fake pass.
- **Unverified optional fields.** A wrong `required` entry blocks legitimate runs with
  `SAFETY_FAIL`; assert required fields only where the API documents them as mandatory.
- **External / non-`$defs` refs.** They resolve to nothing (see §5).
- **Ignored keywords.** `oneOf` / `pattern` look like enforcement and are not.
- **Request schemas.** The validated instance is a *response* document; request bodies
  belong in `references/api-sdk-usage.md`.
