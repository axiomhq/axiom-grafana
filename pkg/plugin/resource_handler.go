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
	mux.HandleFunc("/datasets", d.fetchDatasets)
	mux.HandleFunc("/metrics/", d.fetchDatasets)
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

	j, err := json.Marshal(dsf)
	if err != nil {
		logger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	_, err = w.Write(j)
	if err != nil {
		logger.Error("error writing response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (d *Datasource) fetchDatasets(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	dsf, err := d.api.DatasetFields(context.Background())
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := json.Marshal(dsf)
	if err != nil {
		logger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(j)
	if err != nil {
		logger.Error("error writing response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

	j, err := json.Marshal(dsf)
	if err != nil {
		logger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(j)
	if err != nil {
		logger.Error("error writing response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

	j, err := json.Marshal(dsf)
	if err != nil {
		logger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(j)
	if err != nil {
		logger.Error("error writing response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
