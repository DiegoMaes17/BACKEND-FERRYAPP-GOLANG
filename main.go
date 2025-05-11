package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/database"
	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/handlers"
	"github.com/go-chi/chi/v5"
)

func main() {
	conn, err := database.ConectarBD()
	if err != nil {
		log.Fatal("Error al conectar:", err)
		return
	}
	defer conn.Close(context.Background())

	r := chi.NewRouter()
	//Server log
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Solicitud recibida: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Router

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("¡Funciona!"))
	})

	//Post
	r.Post("/api/empresas/registrar", handlers.RegistrarEmpresa(conn))
	r.Post("/api/empleado/registrar", handlers.RegistrarEmpleado(conn))

	//Put
	r.Put("/api/empresas/actualizar/{rif}", handlers.EditarEmpresas(conn))
	r.Put("/api/empresas/desactivar/{rif}", handlers.EstadoEmpresa(conn))
	r.Put("/api/empresas/activar/{rif}", handlers.EstadoEmpresa(conn))

	r.Put("/api/empleado/actualizar/{cedula}", handlers.EditarEmpleado(conn))
	r.Put("/api/empleado/activar/{cedula}", handlers.EstadoEmpleado(conn))
	r.Put("/api/empleado/desactivar/{cedula}", handlers.EstadoEmpleado(conn))

	// Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor escuchando en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))

}
