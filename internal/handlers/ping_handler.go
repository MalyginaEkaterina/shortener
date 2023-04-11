package handlers

import (
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"net/http"
)

// PingDB is used to check database connection.
func PingDB(store storage.Storage) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		dbStorage, ok := store.(*storage.DBStorage)
		if !ok {
			http.Error(writer, "Failed to check database connection", http.StatusInternalServerError)
		}
		err := dbStorage.Ping(req.Context())
		if err != nil {
			http.Error(writer, "Failed to check database connection", http.StatusInternalServerError)
		}
		writer.WriteHeader(http.StatusOK)
	}
}
