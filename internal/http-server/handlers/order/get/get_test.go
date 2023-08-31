package get_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	mock_cache "test-task/order-service/internal/cache/mocks"
	"test-task/order-service/internal/domain"
	http_server "test-task/order-service/internal/http-server"
	"test-task/order-service/internal/http-server/handlers/order/get"
	mock_get "test-task/order-service/internal/http-server/handlers/order/get/mocks"
	"test-task/order-service/internal/storage"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func Test_GetHandler(t *testing.T) {
	type fields struct {
		cache       *mock_cache.MockCache
		orderGetter *mock_get.MockOrderGetter
	}

	test_cases := []struct {
		test_name  string
		orderId    string
		want       *domain.Order
		fromCache  *domain.Order
		statusCode int
		respErr    string
		prepare    func(f *fields)
	}{
		{
			test_name:  "Success from db",
			orderId:    "b563feb7b2b84b64c8w",
			want:       &domain.Order{OrderUid: "b563feb7b2b84b64c8w"},
			statusCode: 200,
			prepare: func(f *fields) {
				orderId := "b563feb7b2b84b64c8w"
				want := &domain.Order{OrderUid: "b563feb7b2b84b64c8w"}
				gomock.InOrder(
					f.cache.EXPECT().Get(orderId).Return(nil),
					f.orderGetter.EXPECT().Get(gomock.Any(), orderId).Return(want, nil),
					f.cache.EXPECT().Add(orderId, want),
				)
			},
		},
		{
			test_name:  "Success from cache",
			orderId:    "9650f7fa5b404c2f996",
			fromCache:  &domain.Order{OrderUid: "9650f7fa5b404c2f996"},
			statusCode: 200,
			prepare: func(f *fields) {
				orderId := "9650f7fa5b404c2f996"
				fromCache := &domain.Order{OrderUid: "9650f7fa5b404c2f996"}
				f.cache.EXPECT().Get(orderId).Return(fromCache)
			},
		},
		{
			test_name:  "An order with a non-existent ID",
			orderId:    "9650f7fa5b404c2f999",
			respErr:    "not found",
			statusCode: 404,
			prepare: func(f *fields) {
				orderId := "9650f7fa5b404c2f999"
				gomock.InOrder(
					f.cache.EXPECT().Get(orderId).Return(nil),
					f.orderGetter.EXPECT().Get(gomock.Any(), orderId).Return(nil, storage.ErrEntryDoesntExists),
				)
			},
		},
		{
			test_name:  "Internal Error",
			orderId:    "9650f7fa5b404c2f123",
			respErr:    "internal error",
			statusCode: 500,
			prepare: func(f *fields) {
				orderId := "9650f7fa5b404c2f123"
				gomock.InOrder(
					f.cache.EXPECT().Get(orderId).Return(nil),
					f.orderGetter.EXPECT().Get(gomock.Any(), orderId).Return(nil, errors.New("")),
				)
			},
		},
	}

	for i := range test_cases {
		tc := test_cases[i]

		t.Run(tc.test_name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			log := log.Default()

			f := fields{
				cache:       mock_cache.NewMockCache(ctrl),
				orderGetter: mock_get.NewMockOrderGetter(ctrl),
			}

			if tc.prepare != nil {
				tc.prepare(&f)
			}

			router := mux.NewRouter()
			router.HandleFunc("/orders/{order_uid:[a-z0-9]{19}}", get.New(log, f.orderGetter, f.cache)).Methods("GET")

			req := httptest.NewRequest("GET", fmt.Sprintf("/orders/%s", tc.orderId), nil)

			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.statusCode, rec.Code)

			body := rec.Body.String()

			var resp http_server.Response

			assert.NoError(t, json.Unmarshal([]byte(body), &resp))

			assert.Equal(t, tc.respErr, resp.Error)
		})
	}
}
