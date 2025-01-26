package controllers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// writeJSON marshals data structure to encoded JSON response.
func (api *searchAPI) writeJSON(w http.ResponseWriter, status int, data envelope,
	headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	js = append(js, '\n')
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(js); err != nil {
		api.log.Error("failed to write JSON response", zap.Error(err))
		return err
	}

	return nil
}
