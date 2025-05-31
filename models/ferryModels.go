package models

type Ferry struct {
	Matricula          string `json:"matricula"`
	RifEmpresa         string `json:"rif_empresa"`
	Nombre             string `json:"nombre"`
	Modelo             string `json:"modelo"`
	CapacidadEconomica int    `json:"capacidad_economica"`
	CapacidadVIP       int    `json:"capacidad_vip"`
	Estado             bool   `json:"estado"`
}
