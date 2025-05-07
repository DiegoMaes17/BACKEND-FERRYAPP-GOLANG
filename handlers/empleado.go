package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
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
			"Rif_empresa": request.Empleado.Rif_empresa,
			"Nombres":     request.Empleado.Nombres,
			"Cargo":       request.Empleado.Cargo,
			"Apellidos":   request.Empleado.Apellidos,
			"Email":       request.Empleado.Email,
			"Numero_tlf":  request.Empleado.Numero_tlf,
			"Usuario":     request.Usuario.Usuario,
			"Contrasena":  request.Usuario.Contrasena,
			"Cedula":      request.Empleado.Cedula,
		}

		//Campos requeridos
		for field, value := range CamposRequeridos {
			if strings.TrimSpace(value) == "" {
				CamposVacios = append(CamposVacios, field)
			}
		}

		if len(CamposVacios) > 0 {
			MensajeError := "Faltan los siguientes campos:" + strings.Join(CamposVacios, ", ")
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
