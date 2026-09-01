package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"order-service/internal/cache"
	"order-service/internal/store"
)

type Handler struct {
	store *store.Store
	cache *cache.Cache
}

func New(st *store.Store, c *cache.Cache) *Handler {
	return &Handler{store: st, cache: c}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Разрешаем запросы с любого источника (для разработки)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[1] != "order" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	uid := parts[2]

	// Ищем в кеше
	order, found := h.cache.Get(uid)
	if !found {
		var err error
		order, err = h.store.GetOrderByUID(r.Context(), uid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "order not found", http.StatusNotFound)
			} else {
				log.Printf("internal error fetching order %s: %v", uid, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}
		h.cache.Set(uid, order)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("encode error: %v", err)
	}
}
