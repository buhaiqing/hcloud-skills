// Package embed exposes the schema and fixture files compiled into the
// skillcheck binary via //go:embed, so the tool has zero external file
// dependencies at runtime. Mirrors the assets previously living under
// huaweicloud-ces-ops/assets and scripts/fixtures.
package embed

import "embed"

//go:embed schemas/*.json
var schemaFS embed.FS

//go:embed fixtures/*.json
var fixtureFS embed.FS

// Schema files, exposed as byte slices for direct use by validators.
var (
	TraceSchema       = mustRead(schemaFS, "schemas/trace.schema.json")
	SummarySchema     = mustRead(schemaFS, "schemas/summary.schema.json")
	AlarmPlanSchema   = mustRead(schemaFS, "schemas/alarm-plan.schema.json")
	EvalQueriesSchema = mustRead(schemaFS, "schemas/eval-queries.schema.json")
)

// Fixture files used by --self-check secret scans.
var (
	AlarmPlanHealthy = mustRead(fixtureFS, "fixtures/gcl-alarm-plan-healthy.json")
	SummaryHealthy   = mustRead(fixtureFS, "fixtures/gcl-quality-summary-healthy.json")
)

func mustRead(fs embed.FS, name string) []byte {
	data, err := fs.ReadFile(name)
	if err != nil {
		// Embed failures are build-time bugs; panic keeps them loud.
		panic("embed: missing required resource " + name + ": " + err.Error())
	}
	return data
}
