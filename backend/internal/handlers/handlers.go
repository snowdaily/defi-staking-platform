package handlers

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler bundles dependencies (db pool) for the REST handlers.
type Handler struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Handler {
	return &Handler{Pool: pool}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/tvl", h.GetTVL)
	r.Get("/apr", h.GetAPR)
	r.Get("/users/{addr}/position", h.GetUserPosition)
	r.Get("/users/{addr}/history", h.GetUserHistory)
	r.Get("/rewards/recent", h.GetRecentRewards)
	return r
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GET /tvl
func (h *Handler) GetTVL(w http.ResponseWriter, r *http.Request) {
	var totalAssets, totalSupply string
	err := h.Pool.QueryRow(r.Context(), `
        SELECT total_assets::text, total_supply::text
        FROM exchange_rate_snapshots
        ORDER BY timestamp DESC LIMIT 1`).Scan(&totalAssets, &totalSupply)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"totalAssets": "0", "totalSupply": "0"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"totalAssets": totalAssets,
		"totalSupply": totalSupply,
	})
}

// GET /apr — naive 7-day APR from exchange-rate snapshots.
func (h *Handler) GetAPR(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
        SELECT timestamp, rate_e27::text
        FROM exchange_rate_snapshots
        WHERE timestamp >= now() - interval '8 days'
        ORDER BY timestamp ASC`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type pt struct {
		t time.Time
		r *big.Float
	}
	var pts []pt
	for rows.Next() {
		var t time.Time
		var s string
		if err := rows.Scan(&t, &s); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
		if err != nil {
			continue
		}
		pts = append(pts, pt{t, f})
	}

	if len(pts) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"aprPct": 0, "windowDays": 0, "points": len(pts)})
		return
	}
	first, last := pts[0], pts[len(pts)-1]
	if first.r.Sign() == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"aprPct": 0, "windowDays": 0, "points": len(pts)})
		return
	}

	// growth = last/first
	growth := new(big.Float).Quo(last.r, first.r)
	growth.Sub(growth, big.NewFloat(1)) // delta fraction over window
	dt := last.t.Sub(first.t).Hours() / 24
	if dt <= 0 {
		writeJSON(w, http.StatusOK, map[string]any{"aprPct": 0, "windowDays": 0, "points": len(pts)})
		return
	}
	annualised := new(big.Float).Quo(growth, big.NewFloat(dt))
	annualised.Mul(annualised, big.NewFloat(365))
	annualised.Mul(annualised, big.NewFloat(100))

	pct, _ := annualised.Float64()
	writeJSON(w, http.StatusOK, map[string]any{
		"aprPct":     pct,
		"windowDays": dt,
		"points":     len(pts),
	})
}

// GET /users/{addr}/position
func (h *Handler) GetUserPosition(w http.ResponseWriter, r *http.Request) {
	addrStr := chi.URLParam(r, "addr")
	if !common.IsHexAddress(addrStr) {
		writeErr(w, http.StatusBadRequest, "invalid address")
		return
	}
	addr := common.HexToAddress(addrStr)

	var depositsAssets, withdrawalsAssets string
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(assets), 0)::text FROM deposits WHERE owner=$1`, addr.Bytes(),
	).Scan(&depositsAssets); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(assets), 0)::text FROM withdrawals WHERE owner=$1`, addr.Bytes(),
	).Scan(&withdrawalsAssets); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"address":           strings.ToLower(addr.Hex()),
		"totalDeposited":    depositsAssets,
		"totalWithdrawn":    withdrawalsAssets,
	})
}

// GET /users/{addr}/history
func (h *Handler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	addrStr := chi.URLParam(r, "addr")
	if !common.IsHexAddress(addrStr) {
		writeErr(w, http.StatusBadRequest, "invalid address")
		return
	}
	addr := common.HexToAddress(addrStr)

	type entry struct {
		Kind        string    `json:"kind"`
		BlockNumber uint64    `json:"blockNumber"`
		Timestamp   time.Time `json:"timestamp"`
		Assets      string    `json:"assets"`
		Shares      string    `json:"shares"`
	}
	out := []entry{}

	rows, err := h.Pool.Query(r.Context(), `
        SELECT 'deposit', block_number, timestamp, assets::text, shares::text
        FROM deposits WHERE owner=$1
        UNION ALL
        SELECT 'withdraw', block_number, timestamp, assets::text, shares::text
        FROM withdrawals WHERE owner=$1
        ORDER BY 2 DESC LIMIT 100`, addr.Bytes())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Kind, &e.BlockNumber, &e.Timestamp, &e.Assets, &e.Shares); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /rewards/recent
func (h *Handler) GetRecentRewards(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		BlockNumber      uint64    `json:"blockNumber"`
		Timestamp        time.Time `json:"timestamp"`
		Operator         string    `json:"operator"`
		Amount           string    `json:"amount"`
		TotalAssetsAfter string    `json:"totalAssetsAfter"`
	}
	out := []entry{}

	rows, err := h.Pool.Query(r.Context(), `
        SELECT block_number, timestamp, operator, amount::text, total_assets_after::text
        FROM reward_distributions
        ORDER BY block_number DESC LIMIT 50`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var e entry
		var op []byte
		if err := rows.Scan(&e.BlockNumber, &e.Timestamp, &op, &e.Amount, &e.TotalAssetsAfter); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		e.Operator = strings.ToLower(common.BytesToAddress(op).Hex())
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, out)
}
