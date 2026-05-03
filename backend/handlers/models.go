package handlers

import (
	"net/http"
)

type modelsResponse struct {
	Models []string `json:"models"`
}

func (a *Application) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, newAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
		return
	}

	apiKey := APIKeyFromContext(r.Context())
	models, err := a.modelLister.ListModels(r.Context(), apiKey)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, modelsResponse{Models: models})
}
