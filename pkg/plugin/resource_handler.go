package plugin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()
	mux.HandleFunc("/schema-lookup", d.schemaLookup)
	mux.HandleFunc("/metricsdatasets", d.FetchMetricsDatasets)
	mux.HandleFunc("/datasets/{dataset}/metrics", d.fetchDatasetMetrics)
	mux.HandleFunc("/datasets/{dataset}/metrics/{metric}/tags", d.fetchMetricTags)

	return httpadapter.New(mux)
}

func (d *Datasource) schemaLookup(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	dsf, err := d.api.DatasetFields(context.Background())
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) FetchDatasets(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	dsf, err := d.api.DatasetFields(context.Background())
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	datasets := make([]string, 0, len(dsf))
	for _, dataset := range dsf {
		if dataset == nil || dataset.DatasetName == "" {
			continue
		}
		datasets = append(datasets, dataset.DatasetName)
	}

	writeJSON(w, logger, datasets)
}

func (d *Datasource) FetchMetricsDatasets(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	datasets, err := d.api.FetchMetricsDataset(context.Background())
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, datasets)
}

func (d *Datasource) fetchDatasetMetrics(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")

	dsf, err := d.api.GetMetricsForDataset(context.Background(), dataset)
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) fetchMetricTags(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	metric := r.PathValue("metric")

	dsf, err := d.api.GetMetricTags(context.Background(), dataset, metric)
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func writeJSON(w http.ResponseWriter, logger log.Logger, value any) {
	j, err := json.Marshal(value)
	if err != nil {
		logger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(j)
	if err != nil {
		logger.Error("error writing response", "error", err.Error())
		return
	}
}
