package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func ConectarBD() (*pgx.Conn, error) {

	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error cargando .env: %v", err)
	}

	conexionString := os.Getenv("conexion")
	if conexionString == "" {
		return nil, fmt.Errorf("error al obtener el env")
	}

	conn, err := pgx.Connect(context.Background(), conexionString)
	if err != nil {
		return nil, fmt.Errorf("error conecntado a la base de datos: %v", err)
	}

	err = conn.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error al hacer ping a la base de datos: %v", err)
	}

	return conn, nil
}
