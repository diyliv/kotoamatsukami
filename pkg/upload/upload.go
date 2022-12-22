package upload

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
)

var templates = template.Must(template.ParseFiles("templates/upload.html"))

type handler struct {
	logger *zap.Logger
}

func NewHandler(logger *zap.Logger) *handler {
	return &handler{logger: logger}
}

func (h *handler) display(w http.ResponseWriter, page string, data interface{}) {
	templates.ExecuteTemplate(w, page+".html", data)
}

func (h *handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.display(w, "upload", nil)
	case "POST":
		h.Upload(w, r)
	}
}

func (h *handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // max upload 10 mb

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		h.logger.Error("Error while getting file: " + err.Error())
	}
	defer file.Close()

	dst, err := os.Create(fmt.Sprintf("files/%s", handler.Filename))
	if err != nil {
		h.logger.Error("Error while creating file: " + err.Error())
	}

	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.writeResponse(w, http.StatusInternalServerError, "Internal error :(")
		return
	}

	http.Redirect(w, r, "http://localhost:8080/", http.StatusSeeOther)
}

func (h *handler) writeResponse(w http.ResponseWriter, code int, data interface{}) {
	if err := json.NewEncoder(w).Encode(&data); err != nil {
		h.logger.Error("Error while encoding data: " + err.Error())
	}
}
