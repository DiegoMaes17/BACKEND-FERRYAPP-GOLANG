package models

type Empresa struct {
	RIF       string `json:"rif"`
	Nombre    string `json:"nombre"`
	Email     string `json:"email"`
	Direccion string `json:"direccion"`
	Estado    bool   `json:"estado"`
}
