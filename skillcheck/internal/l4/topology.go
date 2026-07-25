package l4

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ResourceType describes a node in the topology graph.
type ResourceType struct {
	Layer       int
	Domain      string
	Description string
}

// ResourceTypes is the static registry of resource type metadata.
var ResourceTypes = map[string]ResourceType{
	"vpc:vpc":            {Layer: 0, Domain: "network", Description: "Virtual Private Cloud"},
	"vpc:subnet":         {Layer: 1, Domain: "network", Description: "Subnet within VPC"},
	"vpc:security_group": {Layer: 1, Domain: "network", Description: "Security Group"},
	"vpc:eip":            {Layer: 2, Domain: "network", Description: "Elastic IP"},
	"ecs:instance":       {Layer: 2, Domain: "compute", Description: "ECS Instance"},
	"ecs:disk":           {Layer: 3, Domain: "storage", Description: "EVS Disk"},
	"elb:loadbalancer":   {Layer: 2, Domain: "network", Description: "Elastic Load Balancer"},
	"elb:listener":       {Layer: 3, Domain: "network", Description: "ELB Listener"},
	"rds:instance":       {Layer: 2, Domain: "database", Description: "RDS Instance"},
	"dcs:instance":       {Layer: 2, Domain: "cache", Description: "DCS Redis Instance"},
	"dns:record":         {Layer: 3, Domain: "network", Description: "DNS Record"},
	"ces:alarm":          {Layer: 4, Domain: "monitoring", Description: "CES Alarm Rule"},
	"iam:policy":         {Layer: 0, Domain: "identity", Description: "IAM Policy"},
	"cbr:vault":          {Layer: 3, Domain: "backup", Description: "CBR Backup Vault"},
	"obs:bucket":         {Layer: 1, Domain: "storage", Description: "OBS Bucket"},
}

// DependencyEdge is a (from, to, relationship) type-level edge.
type DependencyEdge struct {
	From         string
	To           string
	Relationship string
}

// DependencyEdges is the static type-level dependency graph.
var DependencyEdges = []DependencyEdge{
	{"vpc:subnet", "vpc:vpc", "belongs_to"},
	{"vpc:security_group", "vpc:vpc", "belongs_to"},
	{"vpc:eip", "vpc:vpc", "associated_with"},
	{"ecs:instance", "vpc:subnet", "attached_to"},
	{"ecs:instance", "vpc:security_group", "protected_by"},
	{"ecs:instance", "vpc:eip", "bound_to"},
	{"ecs:disk", "ecs:instance", "attached_to"},
	{"elb:loadbalancer", "vpc:subnet", "deployed_in"},
	{"elb:listener", "elb:loadbalancer", "belongs_to"},
	{"elb:loadbalancer", "ecs:instance", "routes_to"},
	{"rds:instance", "vpc:subnet", "deployed_in"},
	{"rds:instance", "vpc:security_group", "protected_by"},
	{"dcs:instance", "vpc:subnet", "deployed_in"},
	{"dns:record", "vpc:eip", "resolves_to"},
	{"dns:record", "elb:loadbalancer", "resolves_to"},
	{"ces:alarm", "ecs:instance", "monitors"},
	{"ces:alarm", "rds:instance", "monitors"},
	{"cbr:vault", "ecs:instance", "backs_up"},
	{"cbr:vault", "rds:instance", "backs_up"},
	{"iam:policy", "ecs:instance", "authorizes"},
	{"iam:policy", "rds:instance", "authorizes"},
}

// SkillResourceMap maps skill → resource type list.
var SkillResourceMap = map[string][]string{
	"huaweicloud-ecs-ops": {"ecs:instance", "ecs:disk"},
	"huaweicloud-vpc-ops": {"vpc:vpc", "vpc:subnet", "vpc:security_group", "vpc:eip"},
	"huaweicloud-elb-ops": {"elb:loadbalancer", "elb:listener"},
	"huaweicloud-rds-ops": {"rds:instance"},
	"huaweicloud-dcs-ops": {"dcs:instance"},
	"huaweicloud-dns-ops": {"dns:record"},
	"huaweicloud-ces-ops": {"ces:alarm"},
	"huaweicloud-cbr-ops": {"cbr:vault"},
	"huaweicloud-iam-ops": {"iam:policy"},
	"huaweicloud-obs-ops": {"obs:bucket"},
}

// TopologyGraph holds nodes + adjacency lists.
type TopologyGraph struct {
	Upstream   map[string][][2]string
	Downstream map[string][][2]string
	Nodes      map[string]map[string]any
}

// NewTopologyGraph constructs an empty graph.
func NewTopologyGraph() *TopologyGraph {
	return &TopologyGraph{
		Upstream:   map[string][][2]string{},
		Downstream: map[string][][2]string{},
		Nodes:      map[string]map[string]any{},
	}
}

// AddNode adds a node to the graph.
func (g *TopologyGraph) AddNode(id, rtype string, metadata map[string]any) {
	rt := ResourceTypes[rtype]
	meta := map[string]any{
		"type":   rtype,
		"layer":  rt.Layer,
		"domain": rt.Domain,
	}
	for k, v := range metadata {
		meta[k] = v
	}
	if _, ok := meta["layer"]; !ok {
		meta["layer"] = 99
	}
	if _, ok := meta["domain"]; !ok {
		meta["domain"] = "unknown"
	}
	g.Nodes[id] = meta
	if _, ok := g.Upstream[id]; !ok {
		g.Upstream[id] = nil
	}
	if _, ok := g.Downstream[id]; !ok {
		g.Downstream[id] = nil
	}
}

// AddEdge adds a directed edge: from depends on to.
func (g *TopologyGraph) AddEdge(from, to, relationship string) {
	g.Upstream[from] = append(g.Upstream[from], [2]string{to, relationship})
	g.Downstream[to] = append(g.Downstream[to], [2]string{from, relationship})
}

// BlastResult is the public return of BlastRadius.
type BlastResult struct {
	Origin            string   `json:"origin"`
	TotalAffected     int      `json:"total_affected"`
	MaxDepthReached   int      `json:"max_depth_reached"`
	AffectedResources []string `json:"affected_resources"`
	DomainsImpacted   []string `json:"domains_impacted"`
}

// BlastRadius walks the downstream graph up to maxDepth, returning all
// affected resources sorted by criticality (highest first).
func (g *TopologyGraph) BlastRadius(origin string, maxDepth int) *BlastResult {
	visited := map[string]bool{}
	type qe struct {
		id    string
		depth int
		rel   string
	}
	queue := []qe{{origin, 0, "origin"}}
	type aff struct {
		id     string
		rtype  string
		domain string
		depth  int
		rel    string
		crit   float64
	}
	var affected []aff
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if visited[c.id] || c.depth > maxDepth {
			continue
		}
		visited[c.id] = true
		if c.depth > 0 {
			info := g.Nodes[c.id]
			rtype, _ := info["type"].(string)
			if rtype == "" {
				rtype = "unknown"
			}
			domain, _ := info["domain"].(string)
			if domain == "" {
				domain = "unknown"
			}
			affected = append(affected, aff{c.id, rtype, domain, c.depth, c.rel, g.Criticality(c.id)})
		}
		for _, e := range g.Downstream[c.id] {
			if !visited[e[0]] {
				queue = append(queue, qe{e[0], c.depth + 1, e[1]})
			}
		}
	}
	sort.SliceStable(affected, func(i, j int) bool { return affected[i].crit > affected[j].crit })
	maxDepthSeen := 0
	resources := make([]string, 0, len(affected))
	domains := map[string]bool{}
	for _, a := range affected {
		if a.depth > maxDepthSeen {
			maxDepthSeen = a.depth
		}
		resources = append(resources, a.id)
		domains[a.domain] = true
	}
	ds := make([]string, 0, len(domains))
	for d := range domains {
		ds = append(ds, d)
	}
	sort.Strings(ds)
	return &BlastResult{
		Origin:            origin,
		TotalAffected:     len(affected),
		MaxDepthReached:   maxDepthSeen,
		AffectedResources: resources,
		DomainsImpacted:   ds,
	}
}

// UpstreamChain walks the upstream graph up to maxDepth.
func (g *TopologyGraph) UpstreamChain(origin string, maxDepth int) []map[string]any {
	visited := map[string]bool{}
	type qe struct {
		id    string
		depth int
		rel   string
	}
	queue := []qe{{origin, 0, "origin"}}
	var out []map[string]any
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if visited[c.id] || c.depth > maxDepth {
			continue
		}
		visited[c.id] = true
		if c.depth > 0 {
			info := g.Nodes[c.id]
			rtype, _ := info["type"].(string)
			out = append(out, map[string]any{
				"resource_id":  c.id,
				"type":         rtype,
				"depth":        c.depth,
				"relationship": c.rel,
			})
		}
		for _, e := range g.Upstream[c.id] {
			if !visited[e[0]] {
				queue = append(queue, qe{e[0], c.depth + 1, e[1]})
			}
		}
	}
	return out
}

// Criticality is downstream_count × layer_weight, where layer_weight = max(1, (5-layer)/5).
func (g *TopologyGraph) Criticality(id string) float64 {
	downstreamCount := len(g.Downstream[id])
	node := g.Nodes[id]
	var layer int
	if v, ok := node["layer"].(int); ok {
		layer = v
	} else {
		layer = 99
	}
	weight := float64(5-layer) / 5.0
	if weight < 1.0 {
		weight = 1.0
	}
	return round2(float64(downstreamCount) * weight)
}

// CriticalPaths returns the top-N most critical resources.
func (g *TopologyGraph) CriticalPaths(topN int) []map[string]any {
	var scored []map[string]any
	for id := range g.Nodes {
		c := g.Criticality(id)
		if c > 0 {
			node := g.Nodes[id]
			rtype, _ := node["type"].(string)
			domain, _ := node["domain"].(string)
			scored = append(scored, map[string]any{
				"resource_id":       id,
				"type":              rtype,
				"domain":            domain,
				"criticality_score": c,
				"direct_dependents": len(g.Downstream[id]),
			})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i]["criticality_score"].(float64) > scored[j]["criticality_score"].(float64)
	})
	if topN > 0 && len(scored) > topN {
		scored = scored[:topN]
	}
	return scored
}

// ToJSON serializes the graph.
func (g *TopologyGraph) ToJSON() map[string]any {
	var edges []map[string]any
	for from, neighbors := range g.Upstream {
		for _, e := range neighbors {
			edges = append(edges, map[string]any{
				"from":         from,
				"to":           e[0],
				"relationship": e[1],
			})
		}
	}
	return map[string]any{
		"schema":       "topology-graph/v1",
		"generated_at": NowISO(),
		"nodes":        g.Nodes,
		"edges":        edges,
		"stats": map[string]any{
			"total_nodes": len(g.Nodes),
			"total_edges": len(edges),
		},
	}
}

// skillNameRE matches `huaweicloud-X-ops` short names in body text.
var skillNameRE = regexp.MustCompile(`huaweicloud-([a-z]+)-ops`)

// fmDelegatesRE matches a `delegates_to:` YAML list in SKILL.md frontmatter.
var fmDelegatesRE = regexp.MustCompile(`(?ms)^delegates_to:\s*\n((?:\s*-\s*huaweicloud-[a-z]+-ops\s*\n)+)`)

// DiscoverDynamicEdges scans SKILL.md frontmatter, body prose, and
// references/integration.md for cross-skill delegation signals. Deduplicates.
func DiscoverDynamicEdges(root string) []DependencyEdge {
	seen := map[string]DependencyEdge{}
	skillDirs, _ := filepath.Glob(filepath.Join(root, "huaweicloud-*-ops"))
	for _, skillDir := range skillDirs {
		primary := filepath.Base(skillDir)
		primaryResources := SkillResourceMap[primary]

		skillMD := filepath.Join(skillDir, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			text := string(data)
			if m := fmDelegatesRE.FindStringSubmatch(text); len(m) >= 2 {
				block := m[1]
				for _, sm := range skillNameRE.FindAllStringSubmatch(block, -1) {
					secondary := "huaweicloud-" + sm[1] + "-ops"
					if secondary == primary {
						continue
					}
					addEdge(seen, primary, secondary, primaryResources, "delegates_to:"+sm[1])
				}
			}
			for _, line := range strings.Split(text, "\n") {
				if !strings.Contains(strings.ToLower(line), "delegate") {
					continue
				}
				for _, sm := range skillNameRE.FindAllStringSubmatch(line, -1) {
					secondary := "huaweicloud-" + sm[1] + "-ops"
					if secondary == primary {
						continue
					}
					addEdge(seen, primary, secondary, primaryResources, "delegates_to:"+sm[1])
				}
			}
		}

		integration := filepath.Join(skillDir, "references", "integration.md")
		f, err := os.Open(integration)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, "huaweicloud-") || !strings.Contains(line, "|") {
				continue
			}
			matches := skillNameRE.FindAllStringSubmatch(line, -1)
			if len(matches) < 2 {
				continue
			}
			secondary := "huaweicloud-" + matches[1][1] + "-ops"
			if secondary == primary {
				continue
			}
			addEdge(seen, primary, secondary, primaryResources, "delegates_to:"+matches[1][1])
		}
		f.Close()
		f2, err := os.Open(integration)
		if err != nil {
			continue
		}
		scanner = bufio.NewScanner(f2)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "|") {
				continue
			}
			for _, sm := range skillNameRE.FindAllStringSubmatch(line, -1) {
				secondary := "huaweicloud-" + sm[1] + "-ops"
				if secondary == primary {
					continue
				}
				addEdge(seen, primary, secondary, primaryResources, "delegates_to:"+sm[1])
			}
		}
		f2.Close()
	}
	out := make([]DependencyEdge, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	return out
}

func addEdge(seen map[string]DependencyEdge, primary, secondary string, primaryResources []string, rel string) {
	secondaryResources := SkillResourceMap[secondary]
	if len(primaryResources) == 0 || len(secondaryResources) == 0 {
		return
	}
	from := primaryResources[0]
	to := secondaryResources[0]
	seen[fmt.Sprintf("%s|%s|%s", from, to, rel)] = DependencyEdge{from, to, rel}
}

// BuildGraphFromSkills assembles a graph from static + (optionally) dynamic edges.
func BuildGraphFromSkills(root string, skills []string, staticOnly bool) *TopologyGraph {
	g := NewTopologyGraph()
	target := skills
	if len(target) == 0 {
		for s := range SkillResourceMap {
			target = append(target, s)
		}
	}
	for _, skill := range target {
		for _, rt := range SkillResourceMap[skill] {
			g.AddNode(rt, rt, map[string]any{"skill": skill})
		}
	}
	for _, e := range DependencyEdges {
		if _, ok := g.Nodes[e.From]; ok {
			if _, ok := g.Nodes[e.To]; ok {
				g.AddEdge(e.From, e.To, e.Relationship)
			}
		}
	}
	if !staticOnly {
		for _, e := range DiscoverDynamicEdges(root) {
			if _, ok := g.Nodes[e.From]; ok {
				if _, ok := g.Nodes[e.To]; ok {
					g.AddEdge(e.From, e.To, e.Relationship)
				}
			}
		}
	}
	return g
}
