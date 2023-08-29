package get

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	http_server "test-task/order-service/internal/http-server"
	"test-task/order-service/internal/schema"
	"test-task/order-service/internal/storage"

	"github.com/gorilla/mux"
)

type OrderGetter interface {
	Get(ctx context.Context, orderId int) (*schema.Order, error)
}

func New(log *log.Logger, orderGetter OrderGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.order.get.New"

		id, err := strconv.Atoi(mux.Vars(r)["id"])

		if err != nil {
			log.Printf("%s: id is incorrect", op)
			RespondOK(http_server.Error("invalid request"), w, r)
			return
		}

		resOrder, err := orderGetter.Get(r.Context(), id)

		if errors.Is(err, storage.ErrEntryDoesntExists) {
			log.Printf("%s: order with id: [%d] not found", op, id)
			RespondOK(http_server.Error("not found"), w, r)
			return
		}

		if err != nil {
			log.Printf("%s: failed to get order with id: [%d] error: %v", op, id, err)
			RespondOK(http_server.Error("internal error"), w, r)
			return
		}

		log.Printf("got order with id: [%d]", id)

		RespondOK(resOrder, w, r)
	}
}

func RespondOK(data any, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}
