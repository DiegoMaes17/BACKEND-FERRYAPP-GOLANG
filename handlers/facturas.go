package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func CrearFactura(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decodificar el JSON de entrada
		var factura models.Factura
		err := json.NewDecoder(r.Body).Decode(&factura)
		if err != nil {
			http.Error(w, "Formato JSON inválido", http.StatusBadRequest)
			return
		}

		// Validar campos obligatorios
		if factura.NombresViajero == "" || factura.ApellidosViajero == "" ||
			factura.RIFEmpresa == "" || factura.CedulaEmpleado == "" ||
			factura.NombreEmpleado == "" || factura.IDViaje == "" ||
			factura.Tipo == "" || factura.MatriculaFerry == "" {
			http.Error(w, "Todos los campos obligatorios son requeridos", http.StatusBadRequest)
			return
		}

		// Establecer valores por defecto
		factura.Estado = true // Estado activo por defecto
		if factura.Emision.IsZero() {
			factura.Emision = time.Now().UTC() // Fecha actual si no se proporciona
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			http.Error(w, "Error iniciando transacción", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		// Insertar nueva factura
		var idFactura int
		err = tx.QueryRow(r.Context(),
			`INSERT INTO facturas (
				nombres_viajero, 
				apellidos_viajero, 
				rif_empresa, 
				cedula_empleado, 
				nombre_empleado, 
				id_viaje, 
				tipo, 
				estado, 
				nota, 
				emision, 
				matricula_ferry
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id_factura`,
			factura.NombresViajero,
			factura.ApellidosViajero,
			factura.RIFEmpresa,
			factura.CedulaEmpleado,
			factura.NombreEmpleado,
			factura.IDViaje,
			factura.Tipo,
			factura.Estado,
			factura.Nota,
			factura.Emision,
			factura.MatriculaFerry,
		).Scan(&idFactura)

		if err != nil {
			http.Error(w, "Error creando factura: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = tx.Commit(r.Context())
		if err != nil {
			http.Error(w, "Error guardando cambios", http.StatusInternalServerError)
			return
		}

		// Respuesta exitosa
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mensaje":    "Factura creada exitosamente",
			"id_factura": idFactura,
		})
	}
}

func ObtenerFactura(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idFactura := chi.URLParam(r, "id")

		var factura models.Factura
		err := db.QueryRow(context.Background(),
			`SELECT 
				id_factura,
				nombres_viajero,
				apellidos_viajero,
				rif_empresa,
				cedula_empleado,
				nombre_empleado,
				id_viaje,
				tipo,
				estado,
				nota,
				emision,
				matricula_ferry
			FROM facturas
			WHERE id_factura = $1`,
			idFactura,
		).Scan(
			&factura.IDFactura,
			&factura.NombresViajero,
			&factura.ApellidosViajero,
			&factura.RIFEmpresa,
			&factura.CedulaEmpleado,
			&factura.NombreEmpleado,
			&factura.IDViaje,
			&factura.Tipo,
			&factura.Estado,
			&factura.Nota,
			&factura.Emision,
			&factura.MatriculaFerry,
		)

		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Factura no encontrada", http.StatusNotFound)
				return
			}
			http.Error(w, "Error al consultar la base de datos", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(factura)
	}
}

func ObtenerFacturasPorEmpresa(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rifEmpresa := r.Context().Value("rif_empresa").(string)

		rows, err := db.Query(r.Context(),
			`SELECT
                id_factura,
                nombres_viajero,
                apellidos_viajero,
                rif_empresa,
                cedula_empleado,
                nombre_empleado,
                id_viaje,
                tipo,
                estado,
                nota,
                emision,
                matricula_ferry
            FROM facturas
            WHERE rif_empresa = $1`,
			rifEmpresa,
		)
		if err != nil {
			http.Error(w, "Error al consultar las facturas: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var facturas []models.Factura
		for rows.Next() {
			var factura models.Factura
			err := rows.Scan(
				&factura.IDFactura,
				&factura.NombresViajero,
				&factura.ApellidosViajero,
				&factura.RIFEmpresa,
				&factura.CedulaEmpleado,
				&factura.NombreEmpleado,
				&factura.IDViaje,
				&factura.Tipo,
				&factura.Estado,
				&factura.Nota,
				&factura.Emision,
				&factura.MatriculaFerry,
			)
			if err != nil {
				http.Error(w, "Error al escanear factura: "+err.Error(), http.StatusInternalServerError)
				return
			}
			facturas = append(facturas, factura)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(facturas)
	}
}

func CambiarEstadoFactura(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idFactura := chi.URLParam(r, "id")
		accion := chi.URLParam(r, "accion")

		var estado bool
		switch accion {
		case "activar":
			estado = true
		case "desactivar":
			estado = false
		default:
			http.Error(w, "Acción no válida. Use 'activar' o 'desactivar'", http.StatusBadRequest)
			return
		}

		// Verificar existencia
		var existe bool
		err := db.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM facturas WHERE id_factura = $1)`,
			idFactura).Scan(&existe)

		if err != nil || !existe {
			http.Error(w, "Factura no encontrada", http.StatusNotFound)
			return
		}

		// Actualizar estado
		_, err = db.Exec(r.Context(),
			`UPDATE facturas SET estado = $1 WHERE id_factura = $2`,
			estado, idFactura)

		if err != nil {
			http.Error(w, "Error actualizando estado: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mensaje":    fmt.Sprintf("Factura %sd correctamente", accion),
			"id_factura": idFactura,
			"estado":     estado,
		})
	}
}
