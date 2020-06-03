# re-prometheus
Reverse engineering of Prometheus text metrics

From metrics from prometheus endpoint to generate markdown about metrics: name, type and labels.

## Run

```
IGNORE_LABELS=sandbox_id go run main.go  http://localhost:8090/metrics
```

This will print out some like this:

```
#### system_load (GAUGE)

Guest system load.

Labels:

  - sandbox_id
    - (depend on env)
  - item
    - load1
    - load15
    - load5
```

`IGNORE_LABELS` means this label is not a fixed cardinality label, need not print label's values.

