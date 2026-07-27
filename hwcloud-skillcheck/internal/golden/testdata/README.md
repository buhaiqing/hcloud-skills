# Golden scenario fixtures

Layout:
  testdata/<skill>/<scenario-name>.json
  testdata/<skill>/<scenario-name>.script.json   (mockhcloud script for the scenario)
  testdata/cross-product/<name>.json
  testdata/cross-product/<name>.script.json

Each `<scenario-name>.json` matches the Scenario struct in
internal/golden/golden.go. The script file follows the scriptlang
schema (entries: [{match, response, exit_code}]).
