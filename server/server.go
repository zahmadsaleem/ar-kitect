package main

import (
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

var (
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

type message struct {
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

func (m *message) receiveFiles() (string, error) {
	namegen := haikunator.New(time.Now().UTC().UnixNano())
	randname := namegen.Haikunate()
	m.FileNames = []string{}
	reader, err := m.FileContent.MultipartReader()
	if err != nil {
		log.Println(err)
		return "something wrong with multipart", err
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			break
		}
		defer part.Close()
		if part.FileName() == "" {
			continue
		}
		thisfname := fmt.Sprintf("%s.%s", randname, filepath.Ext(part.FileName()))
		m.FileNames = append(m.FileNames, thisfname)

		log.Printf("filename: %s", thisfname)
		file, err := os.Create(thisfname)
		if err != nil {
			return "failed to write file", err
		}
		defer file.Close()
		_, _ = io.Copy(file, part)
	}
	return "success", nil
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

func (m *message) writeToFile() (string, error) {

	msg, err := m.receiveFiles()
	if err != nil {
		return msg, err
	}
	return "success", nil
}

func usdzHandler(w http.ResponseWriter, req *http.Request) {
	var t message
	t.FileContent = *req
	t.FileFormat = req.URL.Query().Get("mode")
	if t.FileFormat != OBJ {
		if t.FileFormat != FBX {
			log.Println("mode parameter invalid")
			_, _ = fmt.Fprintf(w, "mode parameter invalid")
			return
		}
	}

	msg, err := t.writeToFile()
	if err != nil {
		log.Println("failed to create obj")
		_, _ = fmt.Fprintln(w, "failed to create obj :"+msg)
		return
	}

	if len(t.FileNames) == 0 {
		log.Println("missing attachments")
		_, _ = fmt.Fprintln(w, "missing attachments")
		return
	}

	fname := t.FileNames[0]

	log.Printf("fname: %s, FileNames : %v, length: %d", fname, t.FileNames, len(t.FileNames))

	// remove received files
	for _, fnm := range t.FileNames {
		defer os.Remove(fnm)
	}

	if t.FileFormat == OBJ {
		ok := convertOBJtoGLTF(w, fname, t)
		fname = ExtractFileNameWithoutExtension(fname)
		if !ok {
			return
		}
	} else if t.FileFormat == FBX {
		ok := convertFBXtoGLTF(w, fname)
		if !ok {
			return
		}
	}
	log.Println("convert to gltf successful")

	ok := convertToUSDZ(w, fname)
	if !ok {
		return
	}

	log.Println("convert to usdz successful")
}

func convertOBJtoGLTF(w http.ResponseWriter, fname string, t message) bool {
	var commandArgs []string
	if !strings.HasSuffix(fname, ".obj") {
		fname = t.FileNames[1]
	}
	log.Println("converting obj file")
	commandArgs = []string{"-i", fname, "-o", fmt.Sprintf("./models/%s", ChangeFileNameExtension(fname, GLTF))}
	_, err := exec.Command(OBJ_TO_GLTF, commandArgs...).Output()
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintln(w, "failed to convert to gltf")
		return false
	}

	return true
}

func convertFBXtoGLTF(w http.ResponseWriter, fname string) bool {
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
		log.Println(string(msg))
		_, _ = fmt.Fprintln(w, "failed to convert to gltf")
		return false
	}
	return true
}

func convertToUSDZ(w http.ResponseWriter, fname string) bool {
	var commandArgs []string
	commandArgs = []string{
		fmt.Sprintf("./models/%s.%s", fname, GLTF),
		fmt.Sprintf("./models/%s.%s", fname, USDZ),
	}
	_, err := exec.Command(GLTF_TO_USDZ, commandArgs...).Output()
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintf(w, "failed to convert to usdz %s\n%s", fname, err)
		return false
	}

	_, _ = fmt.Fprintln(w, fname)
	return true
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

func headers(w http.ResponseWriter, req *http.Request) {
	log.Printf("headers requested")
	for name, headers := range req.Header {
		for _, h := range headers {
			_, _ = fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func createServer(modelsPath string, staticPath string, port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", usdzHandler)
	mux.HandleFunc("/headers", headers)
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

	server := createServer(modelsPath, staticPath, port)

	log.Printf("starting server on port %s", port)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
