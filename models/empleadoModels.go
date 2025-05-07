package models

type Empleados struct {
	Cedula      string `json:"cedula"`
	Nombres     string `json:"nombres"`
	Apellidos   string `json:"apellidos"`
	Rif_empresa string `json:"rif_empresa"`
	Email       string `json:"email"`
	Cargo       string `json:"cargo"`
	Estado      bool   `json:"estado"`
	Numero_tlf  string `json:"numero_tlf"`
}
