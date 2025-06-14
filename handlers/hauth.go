package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/middlewares"
	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func IniciarSesion(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Usuario models.Usuario `json:"usuario"`
		}

		//Enviar errores como JSON
		sendError := func(status int, message string) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": message})
		}

		//Decodificador JSON

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, "Formato JSON invalido", http.StatusBadRequest)
			return
		}

		//Validar campos vacios
		var CamposVacios []string

		CamposRequeridos := map[string]string{
			"usuario":    request.Usuario.Usuario,
			"contrasena": request.Usuario.Contrasena,
		}

		for field, value := range CamposRequeridos {
			if strings.TrimSpace(value) == "" {
				CamposVacios = append(CamposVacios, field)
			}
		}

		if len(CamposVacios) > 0 {
			sendError(http.StatusBadRequest, "Campos vacíos: "+strings.Join(CamposVacios, ", "))
			return
		}
		var (
			usuarioID      string
			contrasenaHash string
			tipoUsuario    string
			estado         bool
		)

		err = db.QueryRow(r.Context(), `SELECT rif_cedula, contrasena, tipo,estado FROM usuarios WHERE usuario = $1`, request.Usuario.Usuario).Scan(&usuarioID, &contrasenaHash, &tipoUsuario, &estado)

		if err != nil {
			if err == pgx.ErrNoRows {
				sendError(http.StatusUnauthorized, "Credenciales inválidas")
			} else {
				sendError(http.StatusInternalServerError, "Error al buscar usuario: "+err.Error())
			}
			return
		}

		//Verificar estado

		if !estado { // Si estado es false
			sendError(http.StatusUnauthorized, "Cuenta inactiva")
			return
		}
		//Verificar contraseña

		if err := bcrypt.CompareHashAndPassword([]byte(contrasenaHash), []byte(request.Usuario.Contrasena)); err != nil {
			sendError(http.StatusUnauthorized, "Credenciales inválidas")
			return
		}

		//Generar JWT
		claims := &middlewares.Claims{
			UsuarioID:   usuarioID,
			TipoUsuario: tipoUsuario,

			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(middlewares.JWTSecret)
		if err != nil {
			sendError(http.StatusInternalServerError, "Error al generar token: "+err.Error())
			return
		}

		//Respuesta extiosa
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"mensaje":    "Autenticación exitosa",
			"token":      tokenStr,
			"tipo":       tipoUsuario,
			"rif_cedula": usuarioID,
		})

	}
}
