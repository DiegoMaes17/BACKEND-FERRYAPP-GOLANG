package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/middlewares"
	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/go-chi/chi/v5"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
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

// Funcion para agregar nuevos usuarios
func RegistrarUsuario(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RifCedula  string `json:"rif_cedula"`
			Usuario    string `json:"usuario"`
			Contrasena string `json:"contrasena"`
			Tipo       string `json:"tipo"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON invalido",
			})
			return
		}

		//Validacion
		if req.RifCedula == "" || req.Usuario == "" || req.Contrasena == "" || req.Tipo == "" {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Todos los campos son requeridos",
			})
			return
		}

		//Hash de contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error procesando credenciales",
			})
			return
		}

		usuario := models.Usuario{
			Rif_Cedula: req.RifCedula,
			Usuario:    req.Usuario,
			Contrasena: string(hashedPassword),
			Tipo:       req.Tipo,
			Estado:     true,
		}

		if err := RegistrarUsuarioTx(r.Context(), db, usuario); err != nil {
			responderError(w, err.(*HandlerError))
			return
		}

		responderJSON(w, http.StatusCreated, map[string]string{
			"mensaje": "Usuario registrado exitosamente",
		})

	}

}

//Funcion de transaccion

func RegistrarUsuarioTx(ctx context.Context, db *pgx.Conn, usuario models.Usuario) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error iniciando transaccion",
		}
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO usuarios (rif_cedula, usuario, contrasena, tipo, estado)
		 VALUES ($1, $2, $3, $4, $5)`,
		usuario.Rif_Cedula,
		usuario.Usuario,
		usuario.Contrasena,
		usuario.Tipo,
		usuario.Estado,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &HandlerError{
				Code:    http.StatusConflict,
				Message: "El usuario o identifacion ya eisten",
			}
		}

		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error al guardar en base de datos",
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error confirmando transaccion",
		}
	}
	return nil

}

// Modificar usuarios (Sin contraseñas)
func EditarUsuario(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifCedula := chi.URLParam(r, "rif_cedula")

		var req struct {
			Usuario    string `json:"usuario"`
			Contrasena string `json:"contrasena"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON invalido",
			})
			return
		}

		//Validar que al menos un campo sea modificado
		if req.Usuario == "" && req.Contrasena == "" {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Debe proporcionar al menos un campo para actulizar",
			})
			return
		}

		//Validar longitud del usuario
		if req.Usuario != "" && (len(req.Usuario) < 4 || len(req.Usuario) > 120) {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "El usuario debe tener entre 4 y 120 caracteres",
			})
			return
		}

		var hashedPassword string
		if req.Contrasena != "" {
			if len(req.Contrasena) < 8 {
				responderError(w, &HandlerError{
					Code:    http.StatusBadRequest,
					Message: "La contraseña debe tener minimo 8 caracteres",
				})
				return
			}

			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
			if err != nil {
				responderError(w, &HandlerError{
					Code:    http.StatusInternalServerError,
					Message: "Error procesando contraseña",
				})
				return
			}
			hashedPassword = string(hashedBytes)
		}
		if err := EditarUsuarioTx(r.Context(), db, rifCedula, req.Usuario, hashedPassword); err != nil {
			responderError(w, err.(*HandlerError))
			return
		}

		responderJSON(w, http.StatusOK, map[string]string{
			"mensaje": "Usuario actulizado exitosamente",
		})
	}
}

// Funcion de transaccion
func EditarUsuarioTx(
	ctx context.Context,
	db *pgx.Conn,
	rifCedula string,
	nuevoUsuario string,
	nuevaContrasena string,
) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error iniciando transacción",
		}
	}
	defer tx.Rollback(ctx)

	// Construcción dinámica de la consulta
	query := "UPDATE usuarios SET"
	params := []interface{}{}
	paramIndex := 1

	updates := []string{}

	if nuevoUsuario != "" {
		updates = append(updates, fmt.Sprintf("usuario = $%d", paramIndex))
		params = append(params, nuevoUsuario)
		paramIndex++
	}

	if nuevaContrasena != "" {
		updates = append(updates, fmt.Sprintf("contrasena = $%d", paramIndex))
		params = append(params, nuevaContrasena)
		paramIndex++
	}

	if len(updates) == 0 {
		return &HandlerError{
			Code:    http.StatusBadRequest,
			Message: "No se proporcionaron campos para actualizar",
		}
	}

	// Agregar condición WHERE
	query += " " + strings.Join(updates, ", ") + fmt.Sprintf(" WHERE rif_cedula = $%d", paramIndex)
	params = append(params, rifCedula)

	// Ejecutar la actualización
	result, err := tx.Exec(ctx, query, params...)
	if err != nil {
		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error ejecutando actualización: " + err.Error(),
		}
	}

	// Verificar si se actualizó algún registro
	if result.RowsAffected() == 0 {
		return &HandlerError{
			Code:    http.StatusNotFound,
			Message: "Usuario no encontrado",
		}
	}

	// Confirmar transacción
	if err := tx.Commit(ctx); err != nil {
		return &HandlerError{
			Code:    http.StatusInternalServerError,
			Message: "Error confirmando cambios: " + err.Error(),
		}
	}

	return nil
}

// Funcion para modificar el estado del usuario (Activado/Desactivado)
func EstadoUsuario(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifCedula := chi.URLParam(r, "rif_cedula")
		accion := chi.URLParam(r, "accion")

		var estado bool
		switch accion {
		case "activar":
			estado = true
		case "desactivar":
			estado = false
		default:
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Acción no válida. Use /activar o /desactivar",
			})
			return
		}

		// Verificar existencia del usuario
		var existe bool
		err := db.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM usuarios WHERE rif_cedula = $1)`,
			rifCedula).Scan(&existe)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error verificando usuario" + err.Error(),
			})
			return
		}

		if !existe {
			responderError(w, &HandlerError{
				Code:    http.StatusNotFound,
				Message: "Usuario no encontrado",
			})
			return
		}

		// Actualización con transacción
		tx, err := db.Begin(r.Context())
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error iniciando transacción",
			})
			return
		}
		defer tx.Rollback(r.Context())

		_, err = tx.Exec(r.Context(),
			`UPDATE usuarios SET estado = $1 WHERE rif_cedula = $2`,
			estado, rifCedula)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error actualizando estado: " + err.Error(),
			})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error confirmando cambios",
			})
			return
		}

		responderJSON(w, http.StatusOK, map[string]interface{}{
			"mensaje": fmt.Sprintf("Usuario %s %s", rifCedula, accion),
			"estado":  estado,
			"accion":  accion,
		})
	}
}

// Funciones para consumos especificos

// Obtener usuarios
func ObtenerUsuario(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifCedula := chi.URLParam(r, "rif_cedula")

		var usuario models.Usuario
		err := db.QueryRow(context.Background(),
			`SELECT rif_cedula, usuario, tipo, estado 
			 FROM usuarios 
			 WHERE rif_cedula = $1`,
			rifCedula,
		).Scan(&usuario.Rif_Cedula, &usuario.Usuario, &usuario.Tipo, &usuario.Estado)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				responderError(w, &HandlerError{
					Code:    http.StatusNotFound,
					Message: "Usuario no encontrado",
				})
				return
			}

			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error al consultar la base de datos",
			})
			return
		}

		responderJSON(w, http.StatusOK, usuario)
	}
}

// Cambio de contraseña (Solo Admin)
func CambiarContrasena(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifCedula := chi.URLParam(r, "rif_cedula")

		var req struct {
			NuevaContrasena string `json:"nuevaContrasena"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON inválido",
			})
			return
		}

		// Validar longitud mínima
		if len(req.NuevaContrasena) < 8 {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "La contraseña debe tener al menos 8 caracteres",
			})
			return
		}

		// Hash de la nueva contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NuevaContrasena), bcrypt.DefaultCost)
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error procesando contraseña",
			})
			return
		}

		// Actualizar en la base de datos
		_, err = db.Exec(context.Background(),
			`UPDATE usuarios SET contrasena = $1 WHERE rif_cedula = $2`,
			string(hashedPassword), rifCedula)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error al actualizar contraseña",
			})
			return
		}

		responderJSON(w, http.StatusOK, map[string]string{
			"mensaje": "Contraseña actualizada exitosamente",
		})
	}
}

// Cambio de contraseña personal
func CambiarContrasenaPersonal(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obtener el ID del usuario del token JWT
		claims := middlewares.UsuarioDesdeContexto(r.Context())
		if claims == nil {
			responderError(w, &HandlerError{
				Code:    http.StatusUnauthorized,
				Message: "No se pudo verificar la identidad del usuario",
			})
			return
		}

		userId := claims.UsuarioID

		var req struct {
			ContrasenaActual string `json:"contrasenaActual"`
			NuevaContrasena  string `json:"nuevaContrasena"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "Formato JSON inválido",
			})
			return
		}

		// Validar longitud mínima
		if len(req.NuevaContrasena) < 8 {
			responderError(w, &HandlerError{
				Code:    http.StatusBadRequest,
				Message: "La nueva contraseña debe tener al menos 8 caracteres",
			})
			return
		}

		// Obtener contraseña actual de la base de datos
		var contrasenaActual string
		err := db.QueryRow(context.Background(),
			`SELECT contrasena FROM usuarios WHERE rif_cedula = $1`,
			userId).Scan(&contrasenaActual)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				responderError(w, &HandlerError{
					Code:    http.StatusNotFound,
					Message: "Usuario no encontrado",
				})
				return
			}

			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error al consultar la base de datos",
			})
			return
		}

		// Verificar contraseña actual
		if err := bcrypt.CompareHashAndPassword([]byte(contrasenaActual), []byte(req.ContrasenaActual)); err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusUnauthorized,
				Message: "Contraseña actual incorrecta",
			})
			return
		}

		// Hash de la nueva contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NuevaContrasena), bcrypt.DefaultCost)
		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error procesando contraseña",
			})
			return
		}

		// Actualizar contraseña
		_, err = db.Exec(context.Background(),
			`UPDATE usuarios SET contrasena = $1 WHERE rif_cedula = $2`,
			string(hashedPassword), userId)

		if err != nil {
			responderError(w, &HandlerError{
				Code:    http.StatusInternalServerError,
				Message: "Error al actualizar contraseña",
			})
			return
		}

		responderJSON(w, http.StatusOK, map[string]string{
			"mensaje": "Contraseña actualizada exitosamente",
		})
	}
}

// Helper para respuesta
func responderError(w http.ResponseWriter, err *HandlerError) {
	responderJSON(w, err.Code, map[string]string{
		"error": err.Message,
	})
}

func responderJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error escribiendo respuesta JSON: %v", err)
		http.Error(w, "Error generando respuesta", http.StatusInternalServerError)
	}
}
