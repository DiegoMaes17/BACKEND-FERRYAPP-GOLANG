package helpers

import (
	"encoding/json"
	"log"
	"net/http"
)

type HandlerError struct {
	Code    int
	Message string
}

func (e *HandlerError) Error() string {
	return e.Message
}

func responderError(w http.ResponseWriter, err *HandlerError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Message,
	})
}

func responderJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error escribiendo respuesta JSON: %v", err)
	}
}
