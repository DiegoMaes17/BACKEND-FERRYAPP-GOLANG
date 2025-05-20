package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var JWTSecret = []byte(os.Getenv("JWTSecret"))

func init() {
	if err := godotenv.Load(); err != nil {
		panic("Error cargando archivo .env")
	}

	JWTSecret := os.Getenv("JWTSecret")
	if JWTSecret == "" {
		panic("JWT_SECRET no esta configurado en las .env")
	}

}

type contextKey string

const (
	//Clave para usuario autenticado
	usuarioContextKey contextKey = "usuario"
)

type Claims struct {
	UsuarioID   string `json:"usuario_id"`
	TipoUsuario string `json:"tipo_usuario"`
	jwt.RegisteredClaims
}

// Helper para acceder al usuario desde el contexto
func UsuarioDesdeContexto(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(usuarioContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// Middleware de autenticacion JWT
func AutenticacionJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			responderError(w, http.StatusUnauthorized, "Token de autorizacion requerido")
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			responderError(w, http.StatusUnauthorized, "Formato de token invalido")
			return
		}

		tokenStr := tokenParts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			responderError(w, http.StatusUnauthorized, "Token invalido o expirado")
			return
		}

		ctx := context.WithValue(r.Context(), usuarioContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//Middleware  de autorizacion para administradores

func SoloAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := UsuarioDesdeContexto(r.Context())
		if claims == nil || claims.TipoUsuario != "Administrador" {
			responderError(w, http.StatusForbidden, "Acceso restringido a administradores")

			return
		}
		next.ServeHTTP(w, r)
	})
}

func responderError(w http.ResponseWriter, status int, mensaje string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": mensaje,
	})
}
