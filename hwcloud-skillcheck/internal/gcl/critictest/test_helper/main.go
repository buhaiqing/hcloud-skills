// critictest/test_helper/main.go — small helper binary for testing
// ExternalCritic. Reads JSON from stdin (drains it so the parent
// close() doesn't wedge), then writes a configurable JSON payload
// from $PAYLOAD_FILE to stdout. exit 0 on success.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	payloadPath := os.Getenv("PAYLOAD_FILE")
	if payloadPath == "" {
		fmt.Fprintln(os.Stderr, "PAYLOAD_FILE unset")
		os.Exit(2)
	}
	// Drain stdin so the parent close() doesn't wedge.
	_, _ = io.ReadAll(os.Stdin)

	data, err := os.ReadFile(payloadPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read payload:", err)
		os.Exit(2)
	}
	if !json.Valid(data) {
		fmt.Fprintln(os.Stderr, "payload is not valid JSON")
		os.Exit(2)
	}
	_, _ = os.Stdout.Write(data)
}
