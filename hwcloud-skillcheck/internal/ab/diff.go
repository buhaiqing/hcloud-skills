package ab

type Result struct {
	PerScenario map[string]string // scenario → stdout excerpt
}

type Diff struct {
	Drift map[string]DriftEntry
}

type DriftEntry struct {
	Old string
	New string
}

func Compare(old, cur Result) *Diff { return CompareWith(old, cur, nil) }

func CompareWith(old, cur Result, allow map[string]bool) *Diff {
	d := &Diff{Drift: map[string]DriftEntry{}}
	for k, oldOut := range old.PerScenario {
		if allow[k] {
			continue
		}
		if curOut, ok := cur.PerScenario[k]; ok && curOut != oldOut {
			d.Drift[k] = DriftEntry{Old: oldOut, New: curOut}
		}
	}
	return d
}

func (d *Diff) HasDrift(scenario string) bool {
	_, ok := d.Drift[scenario]
	return ok
}
