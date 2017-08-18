package doctransport

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeOperationID(t *testing.T) {
	for i, c := range []struct {
		expected string
		method   string
		path     string
	}{
		{"createSession", http.MethodPost, "/sessions"},

		{"getAccounts", http.MethodGet, "/accounts"},
		{"getAccountAddresses", http.MethodGet, "/accounts/{accountID}/addresses"},
		{"getAccountCarts", http.MethodGet, "/accounts/{accountID}/carts"},
		{"createAccountSearchToken", http.MethodPost, "/accounts/{accountID}/search-tokens"},

		{"getUser", http.MethodGet, "/user"},
		{"updateUser", http.MethodPatch, "/user"},
		{"updateUserPassword", http.MethodPut, "/user/password"},

		{"createCart", http.MethodPost, "/carts"},
		{"getCart", http.MethodGet, "/carts/{cartID}"},
		{"updateCartItems", http.MethodPatch, "/carts/{cartID}/items"},

		{"getOrders", http.MethodGet, "/orders"},
		{"createOrder", http.MethodPost, "/orders"},
		{"getOrder", http.MethodGet, "/orders/{orderID}"},
	} {
		t.Run(fmt.Sprintf("%d %s %s", i, c.method, c.path), func(t *testing.T) {
			actual := makeOperationID(c.method, c.path)
			assert.Equal(t, c.expected, actual)
		})
	}
}
