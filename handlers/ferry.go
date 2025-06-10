package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Registrar
func RegistrarFerry(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ferry models.Ferry
		err := json.NewDecoder(r.Body).Decode(&ferry)
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON inválido",
			})
			return
		}

		// Validación de campos obligatorios
		if ferry.Matricula == "" || ferry.RifEmpresa == "" || ferry.Nombre == "" || ferry.Modelo == "" {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Matrícula, RIF empresa, nombre y modelo son requeridos",
			})
			return
		}

		// Validar capacidades
		if ferry.CapacidadEconomica <= 0 || ferry.CapacidadVIP <= 0 {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Las capacidades deben ser mayores a cero",
			})
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error iniciando transacción",
			})
			return
		}
		defer tx.Rollback(r.Context())

		// Verificar existencia previa
		var existe bool
		err = tx.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM ferrys WHERE matricula = $1)`,
			ferry.Matricula,
		).Scan(&existe)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error verificando ferry: " + err.Error(),
			})
			return
		}

		if existe {
			responderError(w, &HandlerError{
				Code:    http.StatusConflict,
				Message: "La matrícula ya está registrada",
			})
			return
		}

		// Insertar ferry
		_, err = tx.Exec(r.Context(),
			`INSERT INTO ferrys (
				matricula, 
				rif_empresa, 
				nombre, 
				modelo, 
				capacidad_economica, 
				capacidad_vip, 
				estado
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			ferry.Matricula,
			ferry.RifEmpresa,
			ferry.Nombre,
			ferry.Modelo,
			ferry.CapacidadEconomica,
			ferry.CapacidadVIP,
			ferry.Estado,
		)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				switch pgErr.Code {
				case "23503":
					responderError(w, &HandlerError{
						Code:    http.StatusBadRequest,
						Message: "El RIF de empresa no existe",
					})
					return
				case "23505":
					responderError(w, &HandlerError{
						Code:    http.StatusConflict,
						Message: "Matrícula duplicada",
					})
					return
				}
			}
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error registrando ferry: " + err.Error(),
			})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error guardando cambios: " + err.Error(),
			})
			return
		}

		responderJSON(w, http.StatusCreated, map[string]string{
			"mensaje":   "Ferry registrado exitosamente",
			"matricula": ferry.Matricula,
		})
	}
}

// EditarFerry actualiza los datos de un ferry existente
func EditarFerry(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		matricula := chi.URLParam(r, "matricula")

		var ferry models.Ferry
		err := json.NewDecoder(r.Body).Decode(&ferry)
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON inválido",
			})
			return
		}

		// Validar capacidades si están presentes
		if ferry.CapacidadEconomica < 0 || ferry.CapacidadVIP < 0 {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Las capacidades no pueden ser negativas",
			})
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error iniciando transacción",
			})
			return
		}
		defer tx.Rollback(r.Context())

		// Verificar existencia
		var existe bool
		err = tx.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM ferrys WHERE matricula = $1)`,
			matricula,
		).Scan(&existe)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error verificando ferry: " + err.Error(),
			})
			return
		}

		if !existe {
			responderError(w, &HandlerError{
				Code:    http.StatusNotFound,
				Message: "Ferry no encontrado",
			})
			return
		}

		// Actualizar campos permitidos (excluyendo matrícula y rif_empresa)
		_, err = tx.Exec(r.Context(),
			`UPDATE ferrys SET 
				nombre = $1,
				modelo = $2,
				capacidad_economica = $3,
				capacidad_vip = $4,
				estado = $5
			WHERE matricula = $6`,
			ferry.Nombre,
			ferry.Modelo,
			ferry.CapacidadEconomica,
			ferry.CapacidadVIP,
			ferry.Estado,
			matricula,
		)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error actualizando ferry: " + err.Error(),
			})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error guardando cambios: " + err.Error(),
			})
			return
		}

		responderJSON(w, http.StatusOK, map[string]string{
			"mensaje":   "Ferry actualizado exitosamente",
			"matricula": matricula,
		})
	}
}

// ObtenerFerry recupera un ferry por su matrícula
func ObtenerFerry(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		matricula := chi.URLParam(r, "matricula")

		var ferry models.Ferry
		err := db.QueryRow(context.Background(),
			`SELECT 
				matricula, 
				rif_empresa, 
				nombre, 
				modelo, 
				capacidad_economica, 
				capacidad_vip, 
				estado 
			FROM ferrys WHERE matricula = $1`,
			matricula,
		).Scan(
			&ferry.Matricula,
			&ferry.RifEmpresa,
			&ferry.Nombre,
			&ferry.Modelo,
			&ferry.CapacidadEconomica,
			&ferry.CapacidadVIP,
			&ferry.Estado,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				responderError(w, &HandlerError{
					Code:    http.StatusNotFound,
					Message: "Ferry no encontrado",
				})
				return
			}
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error en base de datos: " + err.Error(),
			})
			return
		}

		responderJSON(w, http.StatusOK, ferry)
	}
}

func ObtenerFerrysPorEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifEmpresa := chi.URLParam(r, "rif")

		rows, err := db.Query(r.Context(),
			`SELECT matricula, rif_empresa, nombre, modelo, capacidad_economica, capacidad_vip, estado
             FROM ferrys WHERE rif_empresa = $1`, rifEmpresa)
		if err != nil {
			responderError(w, &HandlerError{http.StatusInternalServerError, "Error al buscar ferris"})
			return
		}
		defer rows.Close()

		var ferrys []models.Ferry
		for rows.Next() {
			var f models.Ferry
			if err := rows.Scan(&f.Matricula, &f.RifEmpresa, &f.Nombre, &f.Modelo, &f.CapacidadEconomica, &f.CapacidadVIP, &f.Estado); err != nil {
				responderError(w, &HandlerError{http.StatusInternalServerError, "Error escaneando ferry"})
				return
			}
			ferrys = append(ferrys, f)
		}

		if err = rows.Err(); err != nil {
			responderError(w, &HandlerError{http.StatusInternalServerError, "Error en las filas de ferris"})
			return
		}

		responderJSON(w, http.StatusOK, ferrys)
	}
}

//Codigo hecho de una manera muy rapida por cuestiones de tiempo (Pronta refactorizacion)
