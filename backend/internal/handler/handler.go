package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
		// Из БД
		var err error
		order, err = h.store.GetOrderByUID(r.Context(), uid)
		if err != nil {
			log.Printf("order %s not found: %v", uid, err)
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		// Кладём в кеш для будущих запросов
		h.cache.Set(uid, order)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("encode error: %v", err)
	}
}
