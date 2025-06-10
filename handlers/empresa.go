package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func RegistrarEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Decodificando el JSON
		var request struct {
			Empresa models.Empresa `json:"empresa"`
			Usuario models.Usuario `json:"usuario"`
		}

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, "Formato JSON invalido", http.StatusBadRequest)
			return
		}

		//Validando campos

		if request.Empresa.RIF == "" || request.Empresa.Nombre == "" || request.Empresa.Email == "" || request.Usuario.Usuario == "" || request.Usuario.Contrasena == "" {
			http.Error(w, "Todos los campos son requeridos", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			http.Error(w, "Error iniciando transaccion", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		//Validando existencia
		var rifExistente string
		err = tx.QueryRow(r.Context(),
			`SELECT rif FROM empresa WHERE rif=$1`, request.Empresa.RIF).Scan(&rifExistente)

		if err == nil {
			http.Error(w, `{"error": "Este RIF ya esta registrado"}`, http.StatusConflict)
			return
		}

		// Hash para la contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Usuario.Contrasena), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error procesando contraseña", http.StatusInternalServerError)
			return
		}

		//Insertar  empresa
		_, err = tx.Exec(r.Context(),
			`INSERT INTO empresa (rif, nombre, email, direccion, estado) VALUES ($1, $2, $3, $4, $5)`, request.Empresa.RIF, request.Empresa.Nombre, request.Empresa.Email, request.Empresa.Direccion, true)

		if err != nil {
			//Error RIF
			http.Error(w, "Error registrando empresa: "+err.Error(), http.StatusConflict)
			return
		}

		//Insertar usuario
		_, err = tx.Exec(r.Context(),
			`INSERT INTO usuarios (rif_cedula, usuario, contrasena, tipo, estado) VALUES ($1, $2, $3, $4, $5)`, request.Empresa.RIF, request.Usuario.Usuario, string(hashedPassword), "empresa", true)

		if err != nil {
			http.Error(w, "Error registrando usuario"+err.Error(), http.StatusConflict)
			return
		}

		err = tx.Commit(r.Context())
		if err != nil {
			http.Error(w, "Error guardando cambios", http.StatusInternalServerError)
			return
		}

		//Respuesta exitosa

		w.Header().Set("Content-Type", "aplication/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje": "Registro exitoso",
			"rif":     request.Empresa.RIF,
			"usuario": request.Usuario.Usuario,
		})
	}
}

func EditarEmpresas(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Obteniendo el RIF de la url
		rifParam := chi.URLParam(r, "rif")

		//Decodificando el body
		var empresa models.Empresa
		err := json.NewDecoder(r.Body).Decode(&empresa)
		if err != nil {
			http.Error(w, "Formato JSON invalido", http.StatusBadRequest)
			return
		}

		//Verificando existencia
		var rifExistente string
		err = db.QueryRow(r.Context(),
			`SELECT rif FROM empresa WHERE rif=$1`, rifParam).Scan(&rifExistente)

		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Empresa no encontrada", http.StatusNotFound)
				return
			}
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		//Actulizar empresa
		_, err = db.Exec(r.Context(),
			`UPDATE empresa SET nombre= $1, email= $2, direccion= $3 WHERE rif=$4`, empresa.Nombre, empresa.Email, empresa.Direccion, rifParam)

		if err != nil {
			http.Error(w, "Error actulizando:"+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje": "Empresa actulizada",
		})

	}
}

func EstadoEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Obteniendo el RIF de la url
		rifParam := chi.URLParam(r, "rif")
		estadoParam := chi.URLParam(r, "accion")

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
		var rifExistente string
		err := db.QueryRow(r.Context(),
			`SELECT rif FROM empresa WHERE rif=$1`, rifParam).Scan(&rifExistente)

		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Empresa no encontrada", http.StatusNotFound)
				return
			}
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		//Actulizar
		_, err = db.Exec(r.Context(),
			`UPDATE empresa SET estado= $1 WHERE rif=$2`, estado, rifParam)

		if err != nil {
			http.Error(w, "Error actulizando:"+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje": fmt.Sprintf("Empresa %s correctamente", estadoParam),
			"rif":     rifParam,
			"estado":  fmt.Sprintf("%t", estado),
		})
	}
}

//Obtener empleado por empresa

func EmpleadosPorEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifEmpresa := chi.URLParam(r, "rif")

		rows, err := db.Query(r.Context(),
			`SELECT cedula, nombres, apellidos, rif_empresa, email, cargo, estado, numero_tlf 
             FROM empleados WHERE rif_empresa = $1`, rifEmpresa)
		if err != nil {
			http.Error(w, "Error al buscar empleados: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var empleados []models.Empleados
		for rows.Next() {
			var emp models.Empleados
			if err := rows.Scan(&emp.Cedula, &emp.Nombres, &emp.Apellidos, &emp.Rif_empresa, &emp.Email, &emp.Cargo, &emp.Estado, &emp.Numero_tlf); err != nil {
				http.Error(w, "Error escaneando empleado: "+err.Error(), http.StatusInternalServerError)
				return
			}
			empleados = append(empleados, emp)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, "Error en las filas: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(empleados)
	}
}

//Funciones especficas

// Obtener empresa
func ObtenerEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rif := chi.URLParam(r, "rif")

		var Empresa models.Empresa
		err := db.QueryRow(context.Background(),
			`SELECT rif, nombre, email, direccion,estado 
			 FROM empresa
			 WHERE 	rif = $1`,
			rif,
		).Scan(&Empresa.RIF, &Empresa.Nombre, &Empresa.Email, &Empresa.Direccion, &Empresa.Estado)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				responderError(w, &HandlerError{
					Code:    http.StatusNotFound,
					Message: "Empresa no encontrada",
				})
				return
			}

			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error al consultar la base de datos",
			})
			return
		}

		responderJSON(w, http.StatusOK, Empresa)
	}
}
