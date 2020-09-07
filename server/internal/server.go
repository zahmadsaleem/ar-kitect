package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ar-kitect/server/internal/haikunator"
)

type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type FormFileData struct {
	FileFormat string
	FileNames  []string
}

type middleware struct {
	handler http.Handler
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT")
	m.handler.ServeHTTP(w, r)
}

func newMiddleware(h http.Handler) *middleware {
	return &middleware{h}
}

func (m *FormFileData) receiveFiles(req *http.Request, modelsPath string) (res Result) {
	namegen := haikunator.New(time.Now().UTC().UnixNano())
	randname := namegen.Haikunate()

	reader, err := req.MultipartReader()
	if err != nil {
		res.Message = "something wrong with multipart"
		log.Printf("%s\n%s", res.Message, err)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			res.Message = "error reading multipart"
			log.Printf("%s\n%s", res.Message, err)
			break
		}

		defer part.Close()

		if part.FileName() == "" {
			continue
		}

		thisfname := fmt.Sprintf("%s%s", randname, filepath.Ext(part.FileName()))
		m.FileNames = append(m.FileNames, thisfname)

		log.Printf("filename: %s", thisfname)
		file, err := os.Create(filepath.Join(modelsPath, thisfname))
		if err != nil {
			res.Message = "failed to write file"
			log.Printf("%s\n%s", res.Message, err)
			return
		}
		defer file.Close()

		_, _ = io.Copy(file, part)
	}

	res.Success = true
	return
}

func CreateServer(modelsPath string, staticPath string, port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", convertHandler(modelsPath))
	mux.HandleFunc("/headers", incomingHeadersHandler)
	mux.Handle("/models/", modelsHandler(modelsPath))
	mux.HandleFunc("/", indexHandler(staticPath))
	mux.Handle("/js/", dirHandler(staticPath, "js"))
	mux.Handle("/css/", dirHandler(staticPath, "css"))
	mux.Handle("/img/", dirHandler(staticPath, "img"))
	mainMux := newMiddleware(mux)
	server := &http.Server{
		Addr:    port,
		Handler: mainMux,
	}
	return server
}
