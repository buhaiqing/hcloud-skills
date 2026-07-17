# Knowledge Graph Storage — L5 Root Cause Self-Discovery

> **Purpose**: Implementation of knowledge graph storage using Neo4j or PostgreSQL.
> **Extends**: `knowledge-graph-schema.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Storage Options

| Option | Use Case | Pros | Cons |
|--------|----------|------|------|
| **Neo4j** | Production graph queries | Optimized for graph traversal, Cypher query language | Additional infrastructure |
| **PostgreSQL JSONB** | Simpler setup, hybrid queries | Uses existing PostgreSQL, JSONB indexing | Slower graph traversal |

---

## 2. Neo4j Implementation

### 2.1 Connection

```python
from neo4j import GraphDatabase

class KnowledgeGraphNeo4j:
    def __init__(self, uri, user, password):
        self.driver = GraphDatabase.driver(uri, auth=(user, password))

    def close(self):
        self.driver.close()

    def execute(self, query, params=None):
        with self.driver.session() as session:
            result = session.run(query, params)
            return list(result)
```

### 2.2 Node Operations

```python
def create_alarm_node(kg: KnowledgeGraphNeo4j, alarm_data: dict):
    """
    Create an alarm node in the knowledge graph.
    """
    query = """
    MERGE (a:Alarm {alarm_id: $alarm_id})
    SET a += $properties
    RETURN a
    """
    return kg.execute(query, {
        'alarm_id': alarm_data['alarm_id'],
        'properties': {
            'resource_id': alarm_data['resource_id'],
            'resource_type': alarm_data['resource_type'],
            'severity': alarm_data['severity'],
            'metric_name': alarm_data['metric_name'],
            'threshold': alarm_data['threshold'],
            'triggered_at': alarm_data['triggered_at'].isoformat()
        }
    })

def create_change_node(kg: KnowledgeGraphNeo4j, change_data: dict):
    """
    Create a change node in the knowledge graph.
    """
    query = """
    MERGE (c:Change {change_id: $change_id})
    SET c += $properties
    RETURN c
    """
    return kg.execute(query, {
        'change_id': change_data['change_id'],
        'properties': {
            'change_type': change_data['change_type'],
            'resource_id': change_data['resource_id'],
            'resource_type': change_data['resource_type'],
            'executed_at': change_data['executed_at'].isoformat(),
            'description': change_data['description']
        }
    })
```

### 2.3 Edge Operations

```python
def create_causal_edge(kg: KnowledgeGraphNeo4j, cause_id, effect_id, properties: dict):
    """
    Create a causes edge between two nodes.
    """
    query = """
    MATCH (cause {node_id: $cause_id})
    MATCH (effect {node_id: $effect_id})
    MERGE (cause)-[r:causes]->(effect)
    SET r += $properties
    RETURN r
    """
    return kg.execute(query, {
        'cause_id': cause_id,
        'effect_id': effect_id,
        'properties': properties
    })

def create_resolves_edge(kg: KnowledgeGraphNeo4j, action_id, alarm_id):
    """
    Create a resolves edge between an action and an alarm.
    """
    query = """
    MATCH (action:Action {action_id: $action_id})
    MATCH (alarm:Alarm {alarm_id: $alarm_id})
    MERGE (action)-[r:resolves]->(alarm)
    RETURN r
    """
    return kg.execute(query, {
        'action_id': action_id,
        'alarm_id': alarm_id
    })
```

### 2.4 Query Examples

```python
# Find root causes for an alarm
def find_root_causes(kg: KnowledgeGraphNeo4j, alarm_id: str, depth=3):
    """
    Find potential root causes for an alarm using graph traversal.
    """
    query = """
    MATCH path = (root)-[:causes*1..%d]->(alarm:Alarm {alarm_id: $alarm_id})
    WHERE NOT ()-[:causes]->(root)
    RETURN path, length(path) as depth
    ORDER BY depth
    LIMIT 5
    """ % depth
    return kg.execute(query, {'alarm_id': alarm_id})

# Find all alarms caused by a change
def find_alarms_caused_by_change(kg: KnowledgeGraphNeo4j, change_id: str):
    """
    Find all alarms that were caused by a specific change.
    """
    query = """
    MATCH (change:Change {change_id: $change_id})-[r:causes]->(alarm:Alarm)
    RETURN alarm, r.confidence as confidence, r.time_delta_ms as time_delta_ms
    ORDER BY confidence DESC
    """
    return kg.execute(query, {'change_id': change_id})

# Find correlated alarms
def find_correlated_alarms(kg: KnowledgeGraphNeo4j, alarm_id: str):
    """
    Find alarms that correlate with the given alarm.
    """
    query = """
    MATCH (alarm:Alarm {alarm_id: $alarm_id})-[:correlates_with]-(other:Alarm)
    RETURN other
    """
    return kg.execute(query, {'alarm_id': alarm_id})
```

---

## 3. PostgreSQL JSONB Implementation

### 3.1 Connection

```python
import psycopg2
import json

class KnowledgeGraphPostgres:
    def __init__(self, connection_string):
        self.conn = psycopg2.connect(connection_string)

    def close(self):
        self.conn.close()

    def execute(self, query, params=None):
        with self.conn.cursor() as cursor:
            cursor.execute(query, params)
            if query.strip().upper().startswith('SELECT'):
                return cursor.fetchall()
            else:
                self.conn.commit()
                return cursor.rowcount
```

### 3.2 Node Operations

```python
def create_alarm_node_pg(kg: KnowledgeGraphPostgres, alarm_data: dict):
    """
    Create an alarm node using PostgreSQL JSONB.
    """
    query = """
    INSERT INTO knowledge_graph (node_type, node_id, properties)
    VALUES ('alarm', $1, $2)
    ON CONFLICT (node_type, node_id) DO UPDATE SET properties = $2, updated_at = NOW()
    RETURNING id
    """
    return kg.execute(query, (alarm_data['alarm_id'], json.dumps(alarm_data)))
```

### 3.3 Graph Traversal (Recursive CTE)

```python
def find_root_causes_pg(kg: KnowledgeGraphPostgres, alarm_id: str, max_depth=3):
    """
    Find root causes using recursive CTE (PostgreSQL).
    """
    query = """
    WITH RECURSIVE cause_chain AS (
        -- Base case: start from the target alarm
        SELECT
            g.id,
            g.node_type,
            g.node_id,
            g.properties,
            1 as depth,
            ARRAY[g.id] as path
        FROM knowledge_graph g
        WHERE g.node_type = 'alarm' AND g.node_id = $1

        UNION ALL

        -- Recursive case: follow causes edges
        SELECT
            g.id,
            g.node_type,
            g.node_id,
            g.properties,
            cc.depth + 1,
            cc.path || g.id
        FROM knowledge_graph g
        JOIN knowledge_edges e ON e.target_id = g.id AND e.edge_type = 'causes'
        JOIN cause_chain cc ON cc.id = e.source_id
        WHERE cc.depth < $2
    )
    SELECT * FROM cause_chain ORDER BY depth LIMIT 10
    """
    return kg.execute(query, (alarm_id, max_depth))
```

---

## 4. Query Performance

### 4.1 Indexing Strategy

```sql
-- Neo4j indexes
CREATE INDEX alarm_resource IF NOT EXISTS FOR (a:Alarm) ON (a.resource_id);
CREATE INDEX alarm_time IF NOT EXISTS FOR (a:Alarm) ON (a.triggered_at);
CREATE INDEX change_resource IF NOT EXISTS FOR (c:Change) ON (c.resource_id);
CREATE INDEX cause_confidence IF NOT EXISTS FOR ()-[r:causes]-() ON (r.confidence);

-- PostgreSQL indexes
CREATE INDEX idx_kg_node_type ON knowledge_graph(node_type);
CREATE INDEX idx_kg_node_id ON knowledge_graph(node_id);
CREATE INDEX idx_kg_props_gin ON knowledge_graph USING GIN(properties);
CREATE INDEX idx_edge_source ON knowledge_edges(source_id);
CREATE INDEX idx_edge_target ON knowledge_edges(target_id);
CREATE INDEX idx_edge_type ON knowledge_edges(edge_type);
```

---

## 5. Compliance Checklist

- [ ] Neo4j implementation with Cypher queries
- [ ] PostgreSQL JSONB alternative implementation
- [ ] Node CRUD operations
- [ ] Edge CRUD operations
- [ ] Graph traversal queries (root cause finding)
- [ ] Indexing strategy for performance
