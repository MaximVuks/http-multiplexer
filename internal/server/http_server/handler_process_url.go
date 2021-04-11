package http_server

import (
	"encoding/json"
	"fmt"
	"github.com/maximvuks/http-multiplexer/internal/url_processor"
	"net/http"
)

const urlProcessLimit = 20

type processUrlRequest struct {
	Uri []string `json:"uri"`
}

type processUrlResponse struct {
	Error  string            `json:"error,omitempty"`
	Result map[string]string `json:"result"`
}

type processUrlHandler struct {
	urlProcessor *url_processor.UrlProcessor
}

func NewProcessUrlHandler(urlProcessor *url_processor.UrlProcessor) *processUrlHandler {
	return &processUrlHandler{
		urlProcessor: urlProcessor,
	}
}

func (h *processUrlHandler) ProcessUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method is not supported, use POST", http.StatusNotFound)
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		http.Error(w, "Content-Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	var request processUrlRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(request.Uri) > urlProcessLimit {
		msg := fmt.Sprintf("uri limit exceeded, max: %d", urlProcessLimit)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	response := &processUrlResponse{}
	processedResult, err := h.urlProcessor.ProcessUrls(r.Context(), request.Uri)
	if err != nil {
		response.Error = "an error occurred while processing uri"
	} else {
		response.Result = processedResult
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}
