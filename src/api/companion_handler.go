package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

// staleAfter is when a report is old enough that the page should say so. The
// pages poll every 15s, so this only trips when something is actually wrong.
const staleAfter = 60 * time.Second

// The windows the statistics GUI exposes, plus "all" for the cumulative counts.
// Anything else is refused rather than passed to the game.
var companionWindows = map[string]bool{
	"5s": true, "1m": true, "10m": true, "1h": true,
	"10h": true, "50h": true, "250h": true, "1000h": true, "all": true,
}

// Graph series kinds. A graph always carries both directions, so the kind only
// says which statistics to read from — not which direction.
var companionCategories = map[string]bool{
	"item": true, "fluid": true, "power": true,
}

func CompanionOverviewHandler(w http.ResponseWriter, r *http.Request) {
	writeScopedReport(w, factorio.GetScopedReport("overview"))
}

func CompanionProductionHandler(w http.ResponseWriter, r *http.Request) {
	writeScopedReport(w, factorio.GetScopedReport("production",
		companionSurface(r), companionWindow(r)))
}

func CompanionPowerHandler(w http.ResponseWriter, r *http.Request) {
	writeScopedReport(w, factorio.GetScopedReport("power",
		companionSurface(r), companionWindow(r)))
}

func CompanionScienceHandler(w http.ResponseWriter, r *http.Request) {
	writeScopedReport(w, factorio.GetScopedReport("science",
		companionSurface(r), companionWindow(r)))
}

// CompanionGraphHandler returns the 300 samples behind one series — the data the
// in-game graph draws. It is one series at a time on purpose: 300 points is a
// couple of kilobytes, and a series per row would not fit through RCON.
func CompanionGraphHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if !companionCategories[category] {
		http.Error(w, "unknown graph category: "+category, http.StatusBadRequest)
		return
	}

	series := r.URL.Query().Get("series")
	if series == "" {
		http.Error(w, "a series is required", http.StatusBadRequest)
		return
	}

	writeScopedReport(w, factorio.GetScopedReport("graph",
		companionSurface(r), category, series, companionWindow(r)))
}

// CompanionSpeedHandler applies a game speed from the Controls panel. It means
// a change in the game: the companion mod keeps the new speed across save loads
// and re-applies it after a restart. Admin only.
func CompanionSpeedHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var body struct {
		Speed float64 `json:"speed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "could not read the speed: "+err.Error(), http.StatusBadRequest)
		return
	}

	payload, err := factorio.SetCompanionSpeed(body.Speed)
	if err != nil {
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"speed":     payload["speed"],
		"max_ups":   payload["max_ups"],
		"min_speed": payload["min_speed"],
		"max_speed": payload["max_speed"],
	})
}

func companionSurface(r *http.Request) string {
	return r.URL.Query().Get("surface")
}

func companionWindow(r *http.Request) string {
	window := r.URL.Query().Get("window")

	if !companionWindows[window] {
		return "1h"
	}

	return window
}

func writeScopedReport(w http.ResponseWriter, report *factorio.ScopedReport) {
	age := -1
	if !report.FetchedAt.IsZero() {
		age = int(time.Since(report.FetchedAt).Seconds())
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"scope":       report.Scope,
		"age_seconds": age,
		"stale":       age < 0 || time.Duration(age)*time.Second > staleAfter,
		"error":       report.Error,
		"report":      report.Data,
	}); err != nil {
		log.Printf("Error writing the companion response: %s", err)
	}
}
