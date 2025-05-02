package models

type Usuario struct {
	Rif_Cedula string `json:"rif_cedula"`
	Usuario    string `json:"usuario"`
	Contrasena string `json:"contrasena"`
	Tipo       string `json:"tipo"`
	Estado     bool   `json:"estado"`
}
