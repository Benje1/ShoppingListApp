package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

func Endpoint[T any](
	method string,
	handler func(r *http.Request, input T) (any, error),
) AppHandler {

	return func(w http.ResponseWriter, r *http.Request) (any, error) {

		if r.Method != method {
			return nil, errors.New("method not allowed")
		}

		var input T

		if method == http.MethodPost ||
			method == http.MethodPut ||
			method == http.MethodPatch {

			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				return nil, errors.New("invalid JSON body")
			}
		}

		return handler(r, input)
	}
}

func Post[T any](handler func(r *http.Request, input T) (any, error)) AppHandler {
	return Endpoint(http.MethodPost, handler)
}

func Put[T any](handler func(r *http.Request, input T) (any, error)) AppHandler {
	return Endpoint(http.MethodPut, handler)
}

func Patch[T any](handler func(r *http.Request, input T) (any, error)) AppHandler {
	return Endpoint(http.MethodPatch, handler)
}

func Delete(handler func(r *http.Request) (any, error)) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		if r.Method != http.MethodDelete {
			return nil, errors.New("method not allowed")
		}
		return handler(r)
	}
}

func Get(handler func(r *http.Request) (any, error)) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		if r.Method != http.MethodGet {
			return nil, errors.New("method not allowed")
		}
		return handler(r)
	}
}
