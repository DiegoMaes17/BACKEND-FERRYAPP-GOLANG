package models

import "time"

type Factura struct {
	IDFactura        int       `json:"id_factura"`
	NombresViajero   string    `json:"nombres_viajero"`
	ApellidosViajero string    `json:"apellidos_viajero"`
	RIFEmpresa       string    `json:"rif_empresa"`
	CedulaEmpleado   string    `json:"cedula_empleado"`
	NombreEmpleado   string    `json:"nombre_empleado"`
	IDViaje          string    `json:"id_viaje"`
	Tipo             string    `json:"tipo"`
	Estado           bool      `json:"estado"`
	Nota             string    `json:"nota,omitempty"`
	Emision          time.Time `json:"emision"`
	MatriculaFerry   string    `json:"matricula_ferry"`
}
