package recon

import (
	"net/http"

	// models is imported for swagger annotation resolution (models.APIProblem).
	_ "github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// handleListWiFiClients returns WiFi clients connected to the server's AP/hotspot.
//
//	@Summary		List WiFi hotspot clients
//	@Description	Returns devices connected to the server's WiFi AP/hotspot with signal data
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			ap_device_id	query		string	false	"Filter by AP device ID"
//	@Success		200				{array}		WiFiClientSnapshot
//	@Failure		500				{object}	models.APIProblem
//	@Router			/recon/wifi/clients [get]
func (m *Module) handleListWiFiClients(w http.ResponseWriter, r *http.Request) {
	apDeviceID := r.URL.Query().Get("ap_device_id")

	clients, err := m.store.ListWiFiClients(r.Context(), apDeviceID)
	if err != nil {
		m.logger.Error("failed to list wifi clients", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list wifi clients")
		return
	}
	if clients == nil {
		clients = []WiFiClientSnapshot{}
	}
	writeJSON(w, http.StatusOK, clients)
}
