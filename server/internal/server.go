package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ar-kitect/server/internal/haikunator"
)

type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type FormFileData struct {
	FileFormat  string
	FileNames   []string
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

func (m *FormFileData) receiveFiles(req *http.Request) (res Result) {
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
		file, err := os.Create(thisfname)
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

func ExtractFileNameWithoutExtension(fname string) string {
	split := strings.Split(fname, ".")
	return strings.Join(split[:len(split)-1], ".")
}

func ChangeFileNameExtension(fname string, extn string) string {
	split := strings.Split(fname, ".")
	joined := strings.Join(split[:len(split)-1], ".")
	return fmt.Sprintf("%s.%s", joined, extn)
}

func ConvertHandler(w http.ResponseWriter, req *http.Request) {
	var t FormFileData
	var res Result
	defer func() {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	t.FileFormat = req.URL.Query().Get("mode")
	if t.FileFormat != OBJ && t.FileFormat != FBX {
		res.Message = "mode parameter invalid"
		log.Println(res.Message)
		return
	}

	res = t.receiveFiles(req)
	if !res.Success {
		log.Printf("failed to write obj\n %s", res.Message)
		res.Message = "failed to write obj"
		return
	}

	if len(t.FileNames) == 0 {
		res.Message = "missing attachments"
		log.Println(res.Message)
		return
	}

	fname := t.FileNames[0]
	log.Printf("fname: %s, FileNames : %v, length: %d", fname, t.FileNames, len(t.FileNames))
	// remove received files after conversion
	for _, fnm := range t.FileNames {
		defer os.Remove(fnm)
	}

	if t.FileFormat == OBJ {
		log.Println("converting OBJ to GLTF")
		res = ConvertOBJtoGLTF(fname)
		if !res.Success {
			return
		}
	} else if t.FileFormat == FBX {
		log.Println("converting FBX to GLTF")
		res = ConvertFBXtoGLTF(fname)
		if !res.Success {
			return
		}
	}

	log.Println("convert to GLTF successful")
	fname = ExtractFileNameWithoutExtension(fname)

	res = ConvertToUSDZ(fname)
	if !res.Success {
		return
	}

	log.Println("convert to USDZ successful")
}

func indexHandler(staticPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//if r.URL.Path != "/" {
		//	http.NotFound(w,r)
		//	return
		//}
		http.ServeFile(w, r, staticPath)
	}
}

func modelsHandler(modelsPath string) http.Handler {
	return http.StripPrefix(
		strings.TrimRight(fmt.Sprintf("/models/"), "/"),
		http.FileServer(http.Dir(modelsPath)),
	)
}

func dirHandler(staticPath string, subdir string) http.Handler {
	return http.StripPrefix(
		strings.TrimRight(fmt.Sprintf("/%s/", subdir), "/"),
		http.FileServer(http.Dir(filepath.Join(staticPath, subdir))),
	)
}

func IncomingHeadersHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("headers requested")
	for name, headers := range req.Header {
		for _, h := range headers {
			_, _ = fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func CreateServer(modelsPath string, staticPath string, port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", ConvertHandler)
	mux.HandleFunc("/headers", IncomingHeadersHandler)
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

