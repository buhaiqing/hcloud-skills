# Knowledge Graph Schema — L5 Root Cause Self-Discovery

> **Purpose**: Schema definition for the operational knowledge graph (Neo4j/PostgreSQL).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Node Types

### 1.1 Alarm Node

```yaml
node_type: alarm
properties:
  alarm_id: string (PK)
  alarm_name: string
  resource_id: string
  resource_type: string           # ecs / rds / cce / etc
  severity: string                # critical / major / minor / warning
  metric_name: string
  threshold: float
  triggered_at: timestamp
  resolved_at: timestamp | null
  duration_seconds: int | null
  ces_alarm_id: string
```

### 1.2 Change Node

```yaml
node_type: change
properties:
  change_id: string (PK)
  change_type: string             # deployment / config / scale / manual
  resource_id: string
  resource_type: string
  description: string
  triggered_by: string            # user / system / automated
  executed_at: timestamp
  rollback_available: bool
  cts_trace_id: string
```

### 1.3 Symptom Node

```yaml
node_type: symptom
properties:
  symptom_id: string (PK)
  symptom_type: string            # latency_spike / error_rate / resource_exhaustion
  resource_id: string
  severity: string
  first_observed: timestamp
  last_observed: timestamp
  occurrence_count: int
  description: string
```

### 1.4 Root Cause Node

```yaml
node_type: root_cause
properties:
  cause_id: string (PK)
  cause_category: string          # resource_pressure / config_error / dependency_failure / etc
  cause_type: string              # specific cause type
  description: string
  confidence: float               # 0.0 - 1.0
  first_identified: timestamp
  last_identified: timestamp
  knowledge_source: string        # manual / learned / pattern
  verified: bool
```

---

## 2. Edge Types

### 2.1 causes

```
(alarm) -[causes]-> (alarm)
(alarm) -[causes]-> (symptom)
(alarm) -[causes]-> (root_cause)
(change) -[causes]-> (alarm)
```

```yaml
edge_type: causes
properties:
  confidence: float               # How confident we are in this causation
  time_delta_ms: int              # Typical time between cause and effect
  evidence_count: int             # Number of observed occurrences
  first_observed: timestamp
  last_observed: timestamp
```

### 2.2 triggers

```
(symptom) -[triggers]-> (alarm)
```

### 2.3 correlates_with

```
(alarm) -[correlates_with]-> (alarm)
(symptom) -[correlates_with]-> (symptom)
```

### 2.4 resolves

```
(change) -[resolves]-> (alarm)
(action) -[resolves]-> (alarm)
(root_cause) -[resolves]-> (symptom)
```

### 2.5 precedes

```
(change) -[precedes]-> (alarm)    # Change typically happens before alarm
```

---

## 3. Graph Schema (Cypher / SQL)

### 3.1 Neo4j Schema

```cypher
// Nodes
CREATE CONSTRAINT alarm IF NOT EXISTS
FOR (a:Alarm) REQUIRE a.alarm_id IS UNIQUE;

CREATE CONSTRAINT change IF NOT EXISTS
FOR (c:Change) REQUIRE c.change_id IS UNIQUE;

CREATE CONSTRAINT symptom IF NOT EXISTS
FOR (s:Symptom) REQUIRE s.symptom_id IS UNIQUE;

CREATE CONSTRAINT root_cause IF NOT EXISTS
FOR (r:RootCause) REQUIRE r.cause_id IS UNIQUE;

// Indexes
CREATE INDEX alarm_resource IF NOT EXISTS
FOR (a:Alarm) ON (a.resource_id);

CREATE INDEX alarm_time IF NOT EXISTS
FOR (a:Alarm) ON (a.triggered_at);

CREATE INDEX symptom_type IF NOT EXISTS
FOR (s:Symptom) ON (s.symptom_type);
```

### 3.2 PostgreSQL Schema (JSONB alternative)

```sql
-- Using PostgreSQL JSONB for graph storage
CREATE TABLE knowledge_graph (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type VARCHAR(50) NOT NULL,      -- alarm, change, symptom, root_cause
    node_id VARCHAR(255) NOT NULL,       -- External ID
    properties JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(node_type, node_id)
);

CREATE TABLE knowledge_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES knowledge_graph(id),
    target_id UUID REFERENCES knowledge_graph(id),
    edge_type VARCHAR(50) NOT NULL,      -- causes, triggers, correlates_with, resolves, precedes
    properties JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(source_id, target_id, edge_type)
);

CREATE INDEX idx_node_type ON knowledge_graph(node_type);
CREATE INDEX idx_node_id ON knowledge_graph(node_id);
CREATE INDEX idx_edge_type ON knowledge_edges(edge_type);
CREATE INDEX idx_edge_source ON knowledge_edges(source_id);
CREATE INDEX idx_edge_target ON knowledge_edges(target_id);
```

---

## 4. Sample Data

### 4.1 Alarm Sequence

```cypher
// Alarm: High CPU on ECS instance
CREATE (a1:Alarm {
  alarm_id: 'alarm-001',
  resource_id: 'ecs-12345',
  resource_type: 'ecs',
  severity: 'major',
  metric_name: 'cpu_util',
  threshold: 80.0,
  triggered_at: datetime('2026-07-18T10:00:00Z')
})

// Change: Deployment 30 minutes before
CREATE (c1:Change {
  change_id: 'change-001',
  change_type: 'deployment',
  resource_id: 'ecs-12345',
  executed_at: datetime('2026-07-18T09:30:00Z')
})

// Relationship
CREATE (c1)-[:causes {confidence: 0.85, time_delta_ms: 1800000}]->(a1)
```

---

## 5. Compliance Checklist

- [ ] All 4 node types defined (alarm, change, symptom, root_cause)
- [ ] All 5 edge types defined (causes, triggers, correlates_with, resolves, precedes)
- [ ] Neo4j schema with constraints and indexes
- [ ] PostgreSQL JSONB alternative schema
- [ ] Properties for all nodes and edges
