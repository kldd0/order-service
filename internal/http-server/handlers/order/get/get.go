package get

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"test-task/order-service/internal/cache"
	"test-task/order-service/internal/domain"
	http_server "test-task/order-service/internal/http-server"
	"test-task/order-service/internal/storage"

	"github.com/gorilla/mux"
)

type OrderGetter interface {
	Get(ctx context.Context, orderId string) (*domain.Order, error)
}

func New(log *log.Logger, orderGetter OrderGetter, cache cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.order.get.New"

		uid := mux.Vars(r)["order_uid"]

		if uid == "" {
			log.Printf("%s: id is incorrect", op)
			RespondWithError(errors.New("id is empty"), w, r, "invalid request", http.StatusBadRequest)
			return
		}

		// looking for order in cache
		resOrder := cache.Get(uid)

		if resOrder != nil {
			log.Printf("got order from cache with id: [%s]", uid)
			RespondOK(resOrder, w, r)
			return
		}

		resOrder, err := orderGetter.Get(r.Context(), uid)

		if errors.Is(err, storage.ErrEntryDoesntExists) {
			log.Printf("%s: order with id: [%s] not found", op, uid)
			RespondWithError(err, w, r, "not found", http.StatusNotFound)
			return
		}

		if err != nil {
			log.Printf("%s: failed to get order with id: [%s] error: %v", op, uid, err)
			RespondWithError(err, w, r, "internal error", http.StatusInternalServerError)
			return
		}

		log.Printf("got order with id: [%s]", uid)

		// adding element to cache
		cache.Add(uid, resOrder)
		RespondOK(resOrder, w, r)
	}
}

func RespondOK(data any, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}

func RespondWithError(err error, w http.ResponseWriter, r *http.Request, msg string, status int) {
	log.Printf("error: %s", err)

	resp := http_server.Error(msg)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
