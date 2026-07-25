package l4

import (
	"strings"
	"testing"
)

// --- topology_graph ---

func TestTopologyGraph_AddEdgeAndBlast(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode("vpc:subnet", "vpc:subnet", map[string]any{"skill": "huaweicloud-vpc-ops"})
	g.AddNode("ecs:instance", "ecs:instance", map[string]any{"skill": "huaweicloud-ecs-ops"})
	g.AddEdge("ecs:instance", "vpc:subnet", "attached_to")

	br := g.BlastRadius("vpc:subnet", 3)
	if br.Origin != "vpc:subnet" {
		t.Errorf("origin=%q, want vpc:subnet", br.Origin)
	}
	if br.TotalAffected < 1 {
		t.Errorf("total_affected=%d, want ≥1", br.TotalAffected)
	}
	if !containsString(br.AffectedResources, "ecs:instance") {
		t.Errorf("affected should include ecs:instance, got %+v", br.AffectedResources)
	}
}

func TestTopologyGraph_UpstreamChain(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode("vpc:subnet", "vpc:subnet", nil)
	g.AddNode("ecs:instance", "ecs:instance", nil)
	g.AddEdge("ecs:instance", "vpc:subnet", "attached_to")

	chain := g.UpstreamChain("ecs:instance", 5)
	if len(chain) == 0 {
		t.Fatal("upstream chain should be non-empty")
	}
}

func TestTopologyGraph_Criticality(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode("vpc:subnet", "vpc:subnet", nil)
	g.AddEdge("ecs:instance", "vpc:subnet", "attached_to")
	g.AddEdge("rds:instance", "vpc:subnet", "deployed_in")
	c := g.Criticality("vpc:subnet")
	if c <= 0 {
		t.Errorf("criticality=%v, want >0", c)
	}
}

func TestTopologyGraph_CriticalPaths(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode("vpc:subnet", "vpc:subnet", nil)
	g.AddNode("ecs:instance", "ecs:instance", nil)
	g.AddEdge("ecs:instance", "vpc:subnet", "attached_to")
	paths := g.CriticalPaths(10)
	if len(paths) == 0 {
		t.Fatal("critical paths should be non-empty")
	}
}

func TestTopologyGraph_ToJSON(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode("a", "ecs:instance", nil)
	j := g.ToJSON()
	if j["schema"] != "topology-graph/v1" {
		t.Errorf("schema=%v, want topology-graph/v1", j["schema"])
	}
	stats, _ := j["stats"].(map[string]any)
	if stats == nil {
		t.Fatal("stats missing")
	}
	// In-memory: int; after JSON round-trip: float64. Accept both.
	if v, ok := stats["total_nodes"].(float64); !ok || v != 1 {
		if v2, ok2 := stats["total_nodes"].(int); !ok2 || v2 != 1 {
			t.Errorf("total_nodes=%v (type %T), want 1", stats["total_nodes"], stats["total_nodes"])
		}
	}
}

func TestDiscoverDynamicEdges_FromSkillMD(t *testing.T) {
	root := tmpRepoRoot(t)
	writeSkillWithDelegates(t, root, "ecs", []string{"vpc", "ces"})

	edges := DiscoverDynamicEdges(root)
	if len(edges) == 0 {
		t.Fatal("expected at least one dynamic edge from SKILL.md")
	}
}

func TestBuildGraphFromSkills_StaticAndDynamic(t *testing.T) {
	root := tmpRepoRoot(t)
	writeSkillWithDelegates(t, root, "ecs", []string{"vpc"})

	gStatic := BuildGraphFromSkills(root, nil, true)
	if len(gStatic.Nodes) == 0 {
		t.Error("static graph should have nodes")
	}

	gDyn := BuildGraphFromSkills(root, nil, false)
	if len(gDyn.Nodes) == 0 {
		t.Error("dynamic graph should have nodes")
	}
}

func containsString(s []string, want string) bool {
	for _, x := range s {
		if x == want {
			return true
		}
	}
	return false
}

func tmpRepoRoot(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func writeSkillWithDelegates(t *testing.T, root, short string, delegates []string) {
	t.Helper()
	dir := root + "/huaweicloud-" + short + "-ops"
	mustMkdir(t, dir+"/references")
	yaml := "---\nname: huaweicloud-" + short + "-ops\ndelegates_to:\n"
	for _, d := range delegates {
		yaml += "  - huaweicloud-" + d + "-ops\n"
	}
	yaml += "---\n# skill body\nDelegate to huaweicloud-" + short + "-ops as needed.\n"
	writeFile(t, dir+"/SKILL.md", yaml)
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := writeFileImpl(p, content); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := mkdirAll(p); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

var _ = strings.HasPrefix
