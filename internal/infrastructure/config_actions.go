package infrastructure

import (
	"fmt"
	"net/http"
)

func (api *ConfigAPI) handleScaleUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// scale up logic here...
	fmt.Println("handleScaleUp: scale up", r)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"scaled_up"}`))
}

func (api *ConfigAPI) handleScaleDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// down scale logic here...
	fmt.Println("handleScaleDown: scale down", r)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"scaled_down"}`))
}

func (api *ConfigAPI) handleMorningScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// morning scale logic here...
	fmt.Println("handleMorningScale: scale morning", r)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"morning_scaled"}`))
}

func (api *ConfigAPI) handleEveningScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// evening scaled logic here...
	fmt.Println("handleEveningScale: scale evening", r)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"evening_scaled"}`))
}
