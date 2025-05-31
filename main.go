package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/database"
	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/handlers"
	"github.com/DiegoMaes17/BACKEND-FERRYAPP-GOLANG/middlewares"
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

	//Middleware de logging
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Solicitud recibida: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Solicitud recibida: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	//Ruta publica
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("¡Funciona!"))
	})

	r.Post("/api/login", handlers.IniciarSesion(conn))

	//Grupo de rutas protegidas
	r.Group(func(r chi.Router) {
		//Middleware JWT
		r.Use(middlewares.AutenticacionJWT)

		//Rutas para todos los autenticados
		//Put
		r.Put("/api/empresas/actualizar/{rif}", handlers.EditarEmpresas(conn))
		r.Put("/api/empresas/desactivar/{rif}", handlers.EstadoEmpresa(conn))
		r.Put("/api/empresas/activar/{rif}", handlers.EstadoEmpresa(conn))

		r.Put("/api/empleado/actualizar/{cedula}", handlers.EditarEmpleado(conn))
		r.Put("/api/empleado/activar/{cedula}", handlers.EstadoEmpleado(conn))
		r.Put("/api/empleado/desactivar/{cedula}", handlers.EstadoEmpleado(conn))
		r.Put("/api/usuario/{rif_cedula}", handlers.EditarUsuario(conn))

		//Get
		r.Get("/api/usuario/{rif_cedula}", handlers.ObtenerUsuario(conn))

		//Subgrupo solo para administradores
		r.Group(func(r chi.Router) {
			r.Use(middlewares.SoloAdmin)

			//Rutas de administradores
			//Post
			r.Post("/api/empresas/registrar", handlers.RegistrarEmpresa(conn))
			r.Post("/api/empleado/registrar", handlers.RegistrarEmpleado(conn))
			r.Post("/api/usuario/registrar", handlers.RegistrarUsuario(conn))
			r.Put("/api/usuarios/{rif_cedula}/{accion}", handlers.EstadoUsuario(conn))

		})

	})

	//Server log

	// Router

	// Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor escuchando en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))

}

///Nota 1: Todo el codigo sera refactorizado y mejorado en algun momento.
// Este proyecto es para una asignatura de la univesridad (Desarrollo de software 1)
// Por cuestion de tiempo me veo en la necesidad de apresurar un poco mas el paso y extender los endpoint
// Pero en un futuro planeo retomar el proyecto y mejorar el codigo tanto backend como frontend

// Nota 2: Estoy aprendiendo GO (Golang) en el backend con este proyecto
// Seguramente nadie lea esto :D///
