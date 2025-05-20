package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func RegistrarEmpleado(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Decodificando el JSON

		var request struct {
			Empleado models.Empleados `json:"empleado"`
			Usuario  models.Usuario   `json:"usuario"`
		}

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, "Formato JSON invalido", http.StatusBadRequest)
			return
		}

		//Validadno campos
		var CamposVacios []string

		CamposRequeridos := map[string]string{
			"rif_empresa": request.Empleado.Rif_empresa,
			"nombres":     request.Empleado.Nombres,
			"cargo":       request.Empleado.Cargo,
			"apellidos":   request.Empleado.Apellidos,
			"email":       request.Empleado.Email,
			"numero_tlf":  request.Empleado.Numero_tlf,
			"usuario":     request.Usuario.Usuario,
			"contrasena":  request.Usuario.Contrasena,
			"cedula":      request.Empleado.Cedula,
		}

		//Campos requeridos
		for field, value := range CamposRequeridos {
			if strings.TrimSpace(value) == "" {
				CamposVacios = append(CamposVacios, field)
			}
		}

		if len(CamposVacios) > 0 {
			MensajeError := "Algunos campos estan vacios"
			http.Error(w, MensajeError, http.StatusBadRequest)
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			http.Error(w, "Error iniciando transaccion", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		//Validando existencia de Cedula
		var CedulaExistente string
		err = db.QueryRow(r.Context(),
			`SELECT cedula FROM empleados WHERE cedula=$1`, request.Empleado.Cedula).Scan(&CedulaExistente)

		if err == nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Esta cedula ya esta registrada", http.StatusConflict)
				return
			} else if err != pgx.ErrNoRows {
				http.Error(w, "Error verificando Cedula: ", http.StatusInternalServerError)
				return
			}
		}

		// Hash para la contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Usuario.Contrasena), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error procesando contraseña", http.StatusInternalServerError)
			return
		}
		//Registrando empleado

		_, err = tx.Exec(r.Context(), `INSERT INTO empleados (cedula, nombres, apellidos, rif_empresa, email, cargo, numero_tlf, estado) VALUES ($1, $2, $3, $4, $5, $6, $7,$8 )`, request.Empleado.Cedula, request.Empleado.Nombres, request.Empleado.Apellidos, request.Empleado.Rif_empresa, request.Empleado.Email, request.Empleado.Cargo, request.Empleado.Numero_tlf, true)

		if err != nil {
			http.Error(w, "Error registradno empleado:"+err.Error(), http.StatusConflict)
			return
		}

		//Creando usuario

		_, err = tx.Exec(r.Context(),
			`INSERT INTO usuarios (rif_cedula, usuario, contrasena, tipo, estado) VALUES ($1, $2, $3, $4, $5)`, request.Empleado.Cedula, request.Usuario.Usuario, string(hashedPassword), "empleado", true)

		if err != nil {
			http.Error(w, "Error registrando usuario"+err.Error(), http.StatusConflict)
			return
		}

		err = tx.Commit(r.Context())
		if err != nil {
			http.Error(w, "Error guardadno cambios", http.StatusInternalServerError)
			return
		}

		//Respuesta exitosa

		w.Header().Set("Content-Type", "aplication/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje": "Registro exitoso",
			"Cedula":  request.Empleado.Cedula,
			"Empresa": request.Empleado.Rif_empresa,
			"Usuario": request.Usuario.Usuario,
		})

	}
}

func EditarEmpleado(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Obteniendo Cedula de la URL
		CedulaParam := chi.URLParam(r, "cedula")

		//Decodificando el body
		var empleado models.Empleados
		err := json.NewDecoder(r.Body).Decode(&empleado)
		if err != nil {
			http.Error(w, "Formato JSON invalido", http.StatusBadRequest)
			return
		}

		//Actulizar empleado
		_, err = db.Exec(r.Context(),
			`UPDATE empleados SET nombres=$1 ,apellidos=$2, email=$3, cargo=$4 ,numero_tlf=$5 WHERE cedula=$6`, empleado.Nombres, empleado.Apellidos, empleado.Email, empleado.Cargo, empleado.Numero_tlf, CedulaParam)

		if err != nil {
			http.Error(w, "Error actulizando:"+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje": "Empleado actulizado",
		})
	}
}

func EstadoEmpleado(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Obteniendo la CEDULA de la URL
		CedulaParam := chi.URLParam(r, "cedula")
		estadoParam := strings.Split(r.URL.Path, "/")[3]

		//Estado
		var estado bool
		switch estadoParam {
		case "activar":
			estado = true
		case "desactivar":
			estado = false
		default:
			http.Error(w, "Accion no valida", http.StatusBadRequest)
			return

		}

		//Verificando existencia
		var CedulaExistente string
		err := db.QueryRow(r.Context(),
			`SELECT cedula FROM empleados WHERE cedula=$1`, CedulaParam).Scan(&CedulaExistente)

		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Empleado no encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		//Actulizar
		_, err = db.Exec(r.Context(),
			`UPDATE empleados SET estado= $1 WHERE cedula=$2`, estado, CedulaParam)

		if err != nil {
			http.Error(w, "Error actulizando:"+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Contet-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje":       fmt.Sprintf("Empleado con la cedula %s ,%s correctamente", CedulaParam, estadoParam),
			"Estado actual": fmt.Sprintf("%t", estado),
		})

	}
}
