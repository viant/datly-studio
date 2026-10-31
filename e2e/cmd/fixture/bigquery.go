package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	bq "github.com/viant/bigquery"
	api "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

// This fixture mocks the external BigQuery API, never Studio business handlers.
func startResourceCheckBigQuery(ctx context.Context) error {
	listener, err := net.Listen("tcp", fixtureAddress(18796))
	if err != nil {
		return err
	}
	var mu sync.Mutex
	dry, business, metadata := 0, 0, 0
	jobs := map[string]bool{}
	schema := map[string]any{"fields": []map[string]string{{"name": "id", "type": "STRING", "mode": "NULLABLE"}, {"name": "tenant", "type": "STRING", "mode": "NULLABLE"}, {"name": "name", "type": "STRING", "mode": "NULLABLE"}}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/stats" {
			if r.URL.Query().Get("reset") == "true" {
				mu.Lock()
				dry, business, metadata = 0, 0, 0
				mu.Unlock()
			}
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]int{"dryRuns": dry, "businessQueries": business, "metadataQueries": metadata})
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/jobs") {
			var job api.Job
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&job); err != nil || job.Configuration == nil || job.Configuration.Query == nil {
				http.Error(w, "invalid query job", 400)
				return
			}
			if job.Configuration.DryRun {
				mu.Lock()
				dry++
				mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{"status": map[string]string{"state": "DONE"}, "statistics": map[string]any{"query": map[string]any{"totalBytesProcessed": "2000", "totalBytesProcessedAccuracy": "PRECISE", "statementType": "SELECT", "schema": schema, "referencedTables": []map[string]string{{"projectId": "fixture", "datasetId": "dataset", "tableId": "records"}}}}})
				return
			}
			text := strings.ToUpper(job.Configuration.Query.Query)
			zero := strings.Contains(text, "LIMIT 0") || strings.Contains(text, "1=0") || strings.Contains(text, "1 = 0")
			mu.Lock()
			if zero {
				metadata++
			} else {
				business++
			}
			id := fmt.Sprintf("job-%d", len(jobs)+1)
			jobs[id] = zero
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"jobReference": map[string]string{"projectId": "fixture", "jobId": id}, "status": map[string]string{"state": "DONE"}})
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/queries/") {
			id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			mu.Lock()
			zero, known := jobs[id]
			mu.Unlock()
			if !known {
				http.Error(w, "unknown job", 404)
				return
			}
			rows := []any{}
			count := "0"
			if !zero {
				count = "1"
				rows = []any{map[string]any{"f": []map[string]string{{"v": "row-one"}, {"v": "alpha"}, {"v": "Fixture row"}}}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jobComplete": true, "schema": schema, "totalRows": count, "rows": rows})
			return
		}
		http.Error(w, "unsupported fixture request", 404)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	bq.SetOptions(option.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
	go server.Serve(listener)
	go func() { <-ctx.Done(); _ = server.Shutdown(context.Background()) }()
	return nil
}
