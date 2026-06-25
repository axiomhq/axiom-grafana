package plugin

import (
	"encoding/json"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()
	mux.HandleFunc("/schema-lookup", d.handleSchemaLookup)
	mux.HandleFunc("/metricsdatasets", d.HandleMetricsDatasets)
	mux.HandleFunc("/datasets/{dataset}/metrics", d.handleDatasetMetrics)
	mux.HandleFunc("/datasets/{dataset}/tags", d.handleDatasetTags)
	mux.HandleFunc("/datasets/{dataset}/tags/{tag}/values", d.handleDatasetTagValues)
	mux.HandleFunc("/datasets/{dataset}/metrics/{metric}/tags", d.handleMetricTags)
	mux.HandleFunc("/datasets/{dataset}/metrics/{metric}/tags/{tag}/values", d.handleMetricTagValues)

	return httpadapter.New(mux)
}

func (d *Datasource) handleSchemaLookup(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	dsf, err := d.api.DatasetFields(r.Context())
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) HandleMetricsDatasets(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())

	datasets, err := d.api.FetchMetricsDataset(r.Context())
	if err != nil {
		logger.Error("error listing datasets", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, datasets)
}

func (d *Datasource) handleDatasetMetrics(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	dsf, err := d.api.GetMetricsForDataset(r.Context(), dataset, startTime, endTime)
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) handleDatasetTags(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	dsf, err := d.api.GetMetricTags(r.Context(), dataset, "", startTime, endTime)
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) handleDatasetTagValues(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	tag := r.PathValue("tag")
	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	values, err := d.api.GetMetricTagValues(r.Context(), dataset, "", tag, startTime, endTime)
	if err != nil {
		logger.Error("error looking up tag values", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, values)
}

func (d *Datasource) handleMetricTags(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	metric := r.PathValue("metric")
	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	dsf, err := d.api.GetMetricTags(r.Context(), dataset, metric, startTime, endTime)
	if err != nil {
		logger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, dsf)
}

func (d *Datasource) handleMetricTagValues(w http.ResponseWriter, r *http.Request) {
	logger := log.DefaultLogger.FromContext(r.Context())
	dataset := r.PathValue("dataset")
	metric := r.PathValue("metric")
	tag := r.PathValue("tag")
	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	values, err := d.api.GetMetricTagValues(r.Context(), dataset, metric, tag, startTime, endTime)
	if err != nil {
		logger.Error("error looking up tag values", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, logger, values)
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
