package serverless

import (
	"encoding/json"
	"net/http"
)

// NewErrorHandler returns a func(http.ResponseWriter, *http.Request, error)
// responsible for handling proxy errors
func NewErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusInternalServerError)

		message := "cluster error: " + err.Error()
		msgmap := map[string]string{"error": message}

		json, err := json.Marshal(msgmap)
		if err != nil {
			w.Write([]byte(message))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}
