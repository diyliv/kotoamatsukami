package server

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/diyliv/p2p/pkg/upload"
	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type server struct {
	logger *zap.Logger
}

func NewServer(logger *zap.Logger) *server {
	return &server{
		logger: logger,
	}
}

func (s *server) StartHTTP() {
	color.Magenta("[system] Starting HTTP server on port :8080")

	handler := upload.NewHandler(s.logger)

	router := mux.NewRouter()
	router.HandleFunc("/upload", handler.UploadHandler)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("files")))

	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			s.logger.Error("Error while serving http: " + err.Error())
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	<-done
}
