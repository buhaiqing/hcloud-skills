package learning

import "encoding/json"

// jsonMarshal is a tiny shim used by the test helpers.
func jsonMarshal(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }

// writeJSON is the public test helper (used in *_test.go via package learning).
func writeJSON(p string, v any) error { return writeJSONFile(p, v) }
