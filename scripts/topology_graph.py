#!/usr/bin/env python3
"""Topology Knowledge Graph — resource dependency graph and blast radius analysis.

Maintains a resource dependency graph across Huawei Cloud products, enabling
impact analysis before any mutating operation.

Usage:
  python3 scripts/topology_graph.py build --root . [--skill huaweicloud-ecs-ops]
  python3 scripts/topology_graph.py impact --resource "ecs:instance:i-xxx" [--depth 3]
  python3 scripts/topology_graph.py query --from "vpc:subnet:subnet-xxx" --direction downstream
  python3 scripts/topology_graph.py criticality [--top 10]
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import deque
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

UTC = timezone.utc  # noqa: UP017

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

# --- Resource Type Definitions ---
# Defines known resource types and their dependency relationships
RESOURCE_TYPES: dict[str, dict[str, Any]] = {
    "vpc:vpc": {"layer": 0, "domain": "network", "description": "Virtual Private Cloud"},
    "vpc:subnet": {"layer": 1, "domain": "network", "description": "Subnet within VPC"},
    "vpc:security_group": {"layer": 1, "domain": "network", "description": "Security Group"},
    "vpc:eip": {"layer": 2, "domain": "network", "description": "Elastic IP"},
    "ecs:instance": {"layer": 2, "domain": "compute", "description": "ECS Instance"},
    "ecs:disk": {"layer": 3, "domain": "storage", "description": "EVS Disk"},
    "elb:loadbalancer": {"layer": 2, "domain": "network", "description": "Elastic Load Balancer"},
    "elb:listener": {"layer": 3, "domain": "network", "description": "ELB Listener"},
    "rds:instance": {"layer": 2, "domain": "database", "description": "RDS Instance"},
    "dcs:instance": {"layer": 2, "domain": "cache", "description": "DCS Redis Instance"},
    "dns:record": {"layer": 3, "domain": "network", "description": "DNS Record"},
    "ces:alarm": {"layer": 4, "domain": "monitoring", "description": "CES Alarm Rule"},
    "iam:policy": {"layer": 0, "domain": "identity", "description": "IAM Policy"},
    "cbr:vault": {"layer": 3, "domain": "backup", "description": "CBR Backup Vault"},
    "obs:bucket": {"layer": 1, "domain": "storage", "description": "OBS Bucket"},
}

# --- Static Dependency Edges (type-level) ---
# Format: (from_type, to_type, relationship)
DEPENDENCY_EDGES: list[tuple[str, str, str]] = [
    ("vpc:subnet", "vpc:vpc", "belongs_to"),
    ("vpc:security_group", "vpc:vpc", "belongs_to"),
    ("vpc:eip", "vpc:vpc", "associated_with"),
    ("ecs:instance", "vpc:subnet", "attached_to"),
    ("ecs:instance", "vpc:security_group", "protected_by"),
    ("ecs:instance", "vpc:eip", "bound_to"),
    ("ecs:disk", "ecs:instance", "attached_to"),
    ("elb:loadbalancer", "vpc:subnet", "deployed_in"),
    ("elb:listener", "elb:loadbalancer", "belongs_to"),
    ("elb:loadbalancer", "ecs:instance", "routes_to"),
    ("rds:instance", "vpc:subnet", "deployed_in"),
    ("rds:instance", "vpc:security_group", "protected_by"),
    ("dcs:instance", "vpc:subnet", "deployed_in"),
    ("dns:record", "vpc:eip", "resolves_to"),
    ("dns:record", "elb:loadbalancer", "resolves_to"),
    ("ces:alarm", "ecs:instance", "monitors"),
    ("ces:alarm", "rds:instance", "monitors"),
    ("cbr:vault", "ecs:instance", "backs_up"),
    ("cbr:vault", "rds:instance", "backs_up"),
    ("iam:policy", "ecs:instance", "authorizes"),
    ("iam:policy", "rds:instance", "authorizes"),
]

# --- Skill → Resource Type Mapping ---
SKILL_RESOURCE_MAP: dict[str, list[str]] = {
    "huaweicloud-ecs-ops": ["ecs:instance", "ecs:disk"],
    "huaweicloud-vpc-ops": ["vpc:vpc", "vpc:subnet", "vpc:security_group", "vpc:eip"],
    "huaweicloud-elb-ops": ["elb:loadbalancer", "elb:listener"],
    "huaweicloud-rds-ops": ["rds:instance"],
    "huaweicloud-dcs-ops": ["dcs:instance"],
    "huaweicloud-dns-ops": ["dns:record"],
    "huaweicloud-ces-ops": ["ces:alarm"],
    "huaweicloud-cbr-ops": ["cbr:vault"],
    "huaweicloud-iam-ops": ["iam:policy"],
    "huaweicloud-obs-ops": ["obs:bucket"],
}


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


class TopologyGraph:
    """Resource dependency graph with blast radius analysis."""

    def __init__(self) -> None:
        # Adjacency lists: resource_id → [(neighbor_id, relationship, direction)]
        self.upstream: dict[str, list[tuple[str, str]]] = {}  # what this depends on
        self.downstream: dict[str, list[tuple[str, str]]] = {}  # what depends on this
        self.nodes: dict[str, dict[str, Any]] = {}  # resource metadata

    def add_node(self, resource_id: str, resource_type: str, metadata: dict[str, Any] | None = None) -> None:
        self.nodes[resource_id] = {
            "type": resource_type,
            "layer": RESOURCE_TYPES.get(resource_type, {}).get("layer", 99),
            "domain": RESOURCE_TYPES.get(resource_type, {}).get("domain", "unknown"),
            **(metadata or {}),
        }
        self.upstream.setdefault(resource_id, [])
        self.downstream.setdefault(resource_id, [])

    def add_edge(self, from_id: str, to_id: str, relationship: str) -> None:
        """Add dependency: from_id depends on to_id (to_id is upstream of from_id)."""
        self.upstream.setdefault(from_id, []).append((to_id, relationship))
        self.downstream.setdefault(to_id, []).append((from_id, relationship))

    def blast_radius(self, resource_id: str, max_depth: int = 3) -> dict[str, Any]:
        """Compute blast radius: all downstream resources affected if this fails."""
        affected: list[dict[str, Any]] = []
        visited: set[str] = set()
        queue: deque[tuple[str, int, str]] = deque([(resource_id, 0, "origin")])

        while queue:
            current, depth, rel = queue.popleft()
            if current in visited or depth > max_depth:
                continue
            visited.add(current)

            if depth > 0:
                node_info = self.nodes.get(current, {})
                affected.append({
                    "resource_id": current,
                    "type": node_info.get("type", "unknown"),
                    "domain": node_info.get("domain", "unknown"),
                    "depth": depth,
                    "relationship": rel,
                    "criticality": self._compute_criticality(current),
                })

            for neighbor, relationship in self.downstream.get(current, []):
                if neighbor not in visited:
                    queue.append((neighbor, depth + 1, relationship))

        # Sort by criticality (highest first)
        affected.sort(key=lambda x: -x["criticality"])
        return {
            "origin": resource_id,
            "total_affected": len(affected),
            "max_depth_reached": max(a["depth"] for a in affected) if affected else 0,
            "affected_resources": affected,
            "domains_impacted": sorted(set(a["domain"] for a in affected)),
        }

    def upstream_chain(self, resource_id: str, max_depth: int = 5) -> list[dict[str, Any]]:
        """Trace upstream dependencies (what this resource depends on)."""
        chain: list[dict[str, Any]] = []
        visited: set[str] = set()
        queue: deque[tuple[str, int, str]] = deque([(resource_id, 0, "origin")])

        while queue:
            current, depth, rel = queue.popleft()
            if current in visited or depth > max_depth:
                continue
            visited.add(current)

            if depth > 0:
                node_info = self.nodes.get(current, {})
                chain.append({
                    "resource_id": current,
                    "type": node_info.get("type", "unknown"),
                    "depth": depth,
                    "relationship": rel,
                })

            for neighbor, relationship in self.upstream.get(current, []):
                if neighbor not in visited:
                    queue.append((neighbor, depth + 1, relationship))

        return chain

    def _compute_criticality(self, resource_id: str) -> float:
        """Criticality = number of downstream dependents × layer weight."""
        downstream_count = len(self.downstream.get(resource_id, []))
        node = self.nodes.get(resource_id, {})
        layer = node.get("layer", 99)
        # Lower layer = more foundational = higher criticality
        layer_weight = max(1.0, (5 - layer) / 5.0)
        return round(downstream_count * layer_weight, 2)

    def critical_paths(self, top_n: int = 10) -> list[dict[str, Any]]:
        """Find most critical resources by downstream impact."""
        scored: list[dict[str, Any]] = []
        for rid in self.nodes:
            crit = self._compute_criticality(rid)
            if crit > 0:
                node = self.nodes[rid]
                scored.append({
                    "resource_id": rid,
                    "type": node.get("type", "unknown"),
                    "domain": node.get("domain", "unknown"),
                    "criticality_score": crit,
                    "direct_dependents": len(self.downstream.get(rid, [])),
                })
        scored.sort(key=lambda x: -x["criticality_score"])
        return scored[:top_n]

    def to_json(self) -> dict[str, Any]:
        """Serialize graph to JSON."""
        edges = []
        for from_id, neighbors in self.upstream.items():
            for to_id, rel in neighbors:
                edges.append({"from": from_id, "to": to_id, "relationship": rel})
        return {
            "schema": "topology-graph/v1",
            "generated_at": _now_iso(),
            "nodes": self.nodes,
            "edges": edges,
            "stats": {
                "total_nodes": len(self.nodes),
                "total_edges": len(edges),
            },
        }


def build_graph_from_skills(root: Path, skills: list[str] | None = None) -> TopologyGraph:  # noqa: ARG001
    """Build topology graph from skill definitions and static dependency model.

    Note: `root` is reserved for future dynamic resource discovery from skill assets.
    """
    graph = TopologyGraph()

    # Add all known resource types as nodes
    target_skills = skills or list(SKILL_RESOURCE_MAP.keys())
    for skill in target_skills:
        resource_types = SKILL_RESOURCE_MAP.get(skill, [])
        for rtype in resource_types:
            # Use type as node ID for type-level graph
            graph.add_node(rtype, rtype, {"skill": skill})

    # Add dependency edges (only for nodes in graph)
    for from_type, to_type, rel in DEPENDENCY_EDGES:
        if from_type in graph.nodes and to_type in graph.nodes:
            graph.add_edge(from_type, to_type, rel)

    return graph


def cmd_build(args: argparse.Namespace) -> int:
    """Build topology graph from skill definitions."""
    root: Path = args.root
    skills = args.skill.split(",") if args.skill else None

    graph = build_graph_from_skills(root, skills)
    data = graph.to_json()

    if args.json:
        print(json.dumps(data, indent=2, ensure_ascii=False))
    else:
        print("=== Topology Graph ===")
        print(f"Nodes: {data['stats']['total_nodes']}")
        print(f"Edges: {data['stats']['total_edges']}")
        print()
        print("Resource types:")
        for rid, info in sorted(data["nodes"].items(), key=lambda x: x[1].get("layer", 99)):
            print(f"  L{info.get('layer', '?')} [{info.get('domain', '?')}] {rid}")

    if args.output:
        out = Path(args.output)
        out.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"\nGraph saved: {out}")

    return 0


def cmd_impact(args: argparse.Namespace) -> int:
    """Compute blast radius for a resource."""
    root: Path = args.root
    graph = build_graph_from_skills(root)

    resource = args.resource
    # Allow type-level queries (e.g., "ecs:instance")
    if resource not in graph.nodes:
        print(f"WARN: resource '{resource}' not in graph; available: {sorted(graph.nodes.keys())}", file=sys.stderr)
        return 1

    result = graph.blast_radius(resource, max_depth=args.depth)

    if args.json:
        print(json.dumps(result, indent=2, ensure_ascii=False))
    else:
        print(f"=== Blast Radius: {resource} ===")
        print(f"Total affected: {result['total_affected']}")
        print(f"Max depth: {result['max_depth_reached']}")
        print(f"Domains impacted: {', '.join(result['domains_impacted'])}")
        print()
        for r in result["affected_resources"]:
            indent = "  " * r["depth"]
            print(f"{indent}→ {r['resource_id']} [{r['domain']}] (crit={r['criticality']}, via {r['relationship']})")

    return 0


def cmd_query(args: argparse.Namespace) -> int:
    """Query graph relationships."""
    root: Path = args.root
    graph = build_graph_from_skills(root)

    resource = args.resource_from
    if resource not in graph.nodes:
        print(f"ERROR: resource '{resource}' not in graph", file=sys.stderr)
        return 1

    if args.direction == "upstream":
        results = graph.upstream_chain(resource)
        label = "Upstream dependencies"
    else:
        br = graph.blast_radius(resource, max_depth=5)
        results = br["affected_resources"]
        label = "Downstream dependents"

    if args.json:
        print(json.dumps(results, indent=2, ensure_ascii=False))
    else:
        print(f"=== {label}: {resource} ===")
        for r in results:
            print(f"  {r['resource_id']} [{r.get('type', '?')}] (depth={r['depth']}, via {r['relationship']})")

    return 0


def cmd_criticality(args: argparse.Namespace) -> int:
    """Find most critical resources."""
    root: Path = args.root
    graph = build_graph_from_skills(root)
    critical = graph.critical_paths(top_n=args.top)

    if args.json:
        print(json.dumps(critical, indent=2, ensure_ascii=False))
    else:
        print(f"=== Top {args.top} Critical Resources ===")
        for i, c in enumerate(critical, 1):
            print(f"  {i}. {c['resource_id']} [{c['domain']}] score={c['criticality_score']} dependents={c['direct_dependents']}")

    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Topology Knowledge Graph — resource dependency and blast radius analysis"
    )
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    # build
    b = subparsers.add_parser("build", help="Build topology graph")
    b.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    b.add_argument("--skill", default=None, help="Comma-separated skills to include")
    b.add_argument("--json", action="store_true", help="Output as JSON")
    b.add_argument("--output", default=None, help="Save graph to file")
    b.set_defaults(func=cmd_build)

    # impact
    i = subparsers.add_parser("impact", help="Compute blast radius")
    i.add_argument("--resource", required=True, help="Resource ID or type (e.g., ecs:instance)")
    i.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    i.add_argument("--depth", type=int, default=3, help="Max traversal depth")
    i.add_argument("--json", action="store_true", help="Output as JSON")
    i.set_defaults(func=cmd_impact)

    # query
    q = subparsers.add_parser("query", help="Query graph relationships")
    q.add_argument("--from", dest="resource_from", required=True, help="Source resource")
    q.add_argument("--direction", choices=["upstream", "downstream"], default="downstream")
    q.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    q.add_argument("--json", action="store_true", help="Output as JSON")
    q.set_defaults(func=cmd_query)

    # criticality
    c = subparsers.add_parser("criticality", help="Find most critical resources")
    c.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    c.add_argument("--top", type=int, default=10, help="Number of results")
    c.add_argument("--json", action="store_true", help="Output as JSON")
    c.set_defaults(func=cmd_criticality)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
