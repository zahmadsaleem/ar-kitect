package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ar-kitect/server/haikunator"
)

const (
	OBJ             = "obj"
	FBX             = "fbx"
	GLTF            = "gltf"
	USDZ            = "usdz"
	OBJ_TO_GLTF     = "obj2gltf"
	FBX_TO_GLTF     = "./FBX2glTF"
	GLTF_TO_USDZ    = "usd_from_gltf"
	APP_STATIC_PATH = "APP_STATIC_PATH"
	MODELS_PATH     = "MODELS_PATH"
	SERVER_PORT     = "PORT"
)

type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type FormFileData struct {
	FileFormat  string
	FileContent http.Request
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

func (m *FormFileData) receiveFiles() (res Result) {
	namegen := haikunator.New(time.Now().UTC().UnixNano())
	randname := namegen.Haikunate()
	m.FileNames = []string{}
	reader, err := m.FileContent.MultipartReader()
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
			res.Message ="failed to write file"
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
	t.FileContent = *req
	t.FileFormat = req.URL.Query().Get("mode")
	if t.FileFormat != OBJ && t.FileFormat != FBX {
		res.Message = "mode parameter invalid"
		log.Println(res.Message)
		return
	}

	res = t.receiveFiles()
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

	// remove received files
	for _, fnm := range t.FileNames {
		defer os.Remove(fnm)
	}

	if t.FileFormat == OBJ {
		res = ConvertOBJtoGLTF(fname, t)
		if !res.Success {
			return
		}
	} else if t.FileFormat == FBX {
		res = ConvertFBXtoGLTF(fname)
		if !res.Success {
			return
		}
	}

	log.Println("convert to gltf successful")
	fname = ExtractFileNameWithoutExtension(fname)

	res = ConvertToUSDZ(fname)
	if !res.Success {
		return
	}

	log.Println("convert to usdz successful")
}

func ConvertOBJtoGLTF(fname string, t FormFileData) (res Result) {
	var commandArgs []string
	// TODO: iterate through all files
	if filepath.Ext(fname) == ".obj" {
		fname = t.FileNames[1]
	}
	log.Println("converting OBJ file")
	commandArgs = []string{"-i", fname, "-o", fmt.Sprintf("./models/%s", ChangeFileNameExtension(fname, GLTF))}
	_, err := exec.Command(OBJ_TO_GLTF, commandArgs...).Output()
	if err != nil {
		res.Message = "failed to convert to GLTF"
		log.Printf("%s %s", res.Message, err)
		return
	}

	res.Success = true
	res.Message = "successfully converted OBJ to GLTF"
	return
}

func ConvertFBXtoGLTF(fname string) (res Result) {
	var commandArgs []string
	var msg []byte
	log.Println("converting file format fbx")
	commandArgs = []string{
		"--embed",
		"-i",
		fname,
		"-o",
		fmt.Sprintf("./models/%s", ChangeFileNameExtension(fname, GLTF)),
	}
	msg, err := exec.Command(FBX_TO_GLTF, commandArgs...).Output()
	if err != nil {
		log.Printf("FBX_TO_GLTF error, %s\n", string(msg))
		res.Message = "failed to convert to gltf"
		return
	}
	res.Success = true
	return
}

func ConvertToUSDZ(fname string) (res Result) {
	var commandArgs []string
	commandArgs = []string{
		fmt.Sprintf("./models/%s.%s", fname, GLTF),
		fmt.Sprintf("./models/%s.%s", fname, USDZ),
	}
	_, err := exec.Command(GLTF_TO_USDZ, commandArgs...).Output()
	if err != nil {
		log.Printf("failed to convert to usdz %s\n%s", fname, err)
		res.Message = fmt.Sprintf("failed to convert `%s` to usdz", fname)
		return
	}

	res.Success = true
	res.Message = fname
	return
}

func pathsMustExist(paths ...string) {
	for _, p := range paths {
		abspath, _ := filepath.Abs(p)
		if _, err := os.Stat(p); os.IsNotExist(err) || p == "" {
			panic(fmt.Sprintf("path '%s' is empty or not accessible", abspath))
		}
		log.Println(p)
	}
}

func indexHandler(staticPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

func main() {
	port := fmt.Sprintf(":%s", os.Getenv(SERVER_PORT))
	staticPath, _ := os.LookupEnv(APP_STATIC_PATH)
	modelsPath, _ := os.LookupEnv(MODELS_PATH)

	pathsMustExist(staticPath, modelsPath)
	log.Printf("static path %s, models path %s", staticPath, modelsPath)

	server := CreateServer(modelsPath, staticPath, port)

	log.Printf("starting server on port %s", port)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
