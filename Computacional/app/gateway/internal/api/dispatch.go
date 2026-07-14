// internal/api/dispatch.go
//
// POST /api/orders/{id}/dispatch — resolves a named waypoint and dispatches
// the order via OrderService.Dispatch.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"unbot-gateway/internal/services"
)

// ── Request type ──────────────────────────────────────────────────────────────

// dispatchRequest carries the named delivery point. Coordinates are resolved
// server-side from the calibrated delivery-point source.
type dispatchRequest struct {
	RestaurantName string `json:"restaurant_name"`
	WaypointName   string `json:"waypoint_name"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

func (s *Server) dispatchHandler(orderSvc *services.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed,
				errorResponse{Error: "method not allowed"})
			return
		}

		orderID := strings.TrimSpace(r.PathValue("id"))
		if orderID == "" {
			writeJSON(w, http.StatusBadRequest,
				errorResponse{Error: "order_id path parameter is required"})
			return
		}

		var req dispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest,
				errorResponse{Error: "malformed JSON body"})
			return
		}

		// ── Build Destination from request ────────────────────────────────
		if req.WaypointName == "" {
			writeJSON(w, http.StatusBadRequest,
				errorResponse{Error: "a calibrated delivery point is required"})
			return
		}

		dest := services.Destination{WaypointName: req.WaypointName}

		// ── Delegate to service ───────────────────────────────────────────
		result, err := orderSvc.Dispatch(r.Context(), orderID, dest)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrUnknownWaypoint):
				s.log.Warn("dispatch rejected: unknown waypoint",
					"order_id", orderID,
					"waypoint", req.WaypointName,
				)
				writeJSON(w, http.StatusBadRequest,
					errorResponse{Error: "unknown destination waypoint: " + req.WaypointName})
			case errors.Is(err, services.ErrOTPIssuance):
				s.log.Error("OTP issuance failed in dispatch",
					"order_id", orderID,
					"error", err,
				)
				writeJSON(w, http.StatusInternalServerError,
					errorResponse{Error: "failed to generate order code; please retry"})
			default:
				s.log.Error("unexpected error in Dispatch",
					"order_id", orderID,
					"error", err,
				)
				writeJSON(w, http.StatusInternalServerError,
					errorResponse{Error: "internal server error"})
			}
			return
		}

		s.log.Info("order dispatched",
			"order_id", orderID,
			"restaurant", req.RestaurantName,
			"waypoint", result.WaypointName,
			"gateway_mode", result.GatewayMode,
			"mqtt_connected", result.MQTTConnected,
		)

		writeJSON(w, http.StatusOK, result)
	}
}
