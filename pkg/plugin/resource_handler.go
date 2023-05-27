package plugin

import (
	"context"
	"encoding/json"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()
	mux.Handle("/schema-lookup", d)

	return httpadapter.New(mux)
}

func (d *Datasource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	dsf, err := d.DatasetFields(context.Background())
	if err != nil {
		log.DefaultLogger.Error("error looking up schema", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := json.Marshal(dsf)
	if err != nil {
		log.DefaultLogger.Error("error marshaling json", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	_, err = w.Write(j)
	if err != nil {
		log.DefaultLogger.Error("error writing response", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
