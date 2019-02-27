package fixture

import (
	"encoding/json"
	"net/http"
)

func MockHTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v4/pages/domain" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	host := r.FormValue("host")
	config, ok := internalConfigs[host]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&config)
}
