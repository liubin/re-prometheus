# re-prometheus
Reverse engineering of Prometheus text metrics

From metrics from prometheus endpoint to generate markdown about metrics: name, type and labels.

## Run

```
UNFIXED_LABELS=sandbox_id,device,interface,disk go run main.go  http://localhost:8090/metrics
```

This will write metrics yaml and markdown file under `tmp/`.

`UNFIXED_LABELS` means this label is not a fixed cardinality label, need not print label's values.

