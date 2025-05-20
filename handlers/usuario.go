package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
)

// Esta funcion solo sera usada para administradores, el resto de tipo de usuarios seran creados junto a sus datos relacionados (Empresa/Empleado)

// Custom error type para manejo especifico de errores
type HandlerError struct {
	Code    int
	Message string
}

func (e *HandlerError) Error() string {
	return e.Message
}

// Request struct para validacion y JSON
type UsuarioRequest struct {
	Usuario models.Usuario `json:"usuario"`
}

// func RegistrarUsuario(db *pgx.Conn) http.HandlerFunc {
// validate := validator.New()

// }

// }

//Helpers para respuestas

func responderError(w http.ResponseWriter, err *HandlerError) {
	responderJSON(w, err.Code, map[string]string{
		"error": err.Message,
	})
}
func responderJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Contet-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error escribiendo respuesta JSON: %v", err)
		http.Error(w, "Error generando respuesta", http.StatusInternalServerError)
	}
}
