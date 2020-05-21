package main

import (
	"crypto/tls"
	"fmt"

	// "haikunator"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
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

func expireFiles(fnames []string) {
	time.Sleep(1 * time.Hour)
	var fname string
	for _, f := range fnames {
		fname = "./models/" + f
		if _, err := os.Stat(fname); err != nil {
			os.Remove(fname)
			fmt.Println(fname)
		}
	}
}

func (m *message) receiveFiles() (string, error) {
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
		} else if err != nil {
			break
		}
		defer part.Close()
		if part.FileName() == "" {
			continue
		}
		m.FileNames = append(m.FileNames, part.FileName())
		// s := haikunator.Haikunate()
		log.Println("filename: " + part.FileName())
		d, err := os.Create(part.FileName())
		if err != nil {
			// log.Fatal(err)
			return "failed to write file", err
		}
		defer d.Close()
		io.Copy(d, part)
	}
	return "success", nil
}

func (m *message) writeToFile() (string, error) {

	msg, err := m.receiveFiles()
	if err != nil {
		return msg, err
	}
	return "success", nil
}

func usdz(w http.ResponseWriter, req *http.Request) {
	var t message
	var err error
	t.FileContent = *req
	t.FileFormat = req.URL.Query().Get("mode")
	if t.FileFormat != "obj" {
		if t.FileFormat != "fbx" {
			log.Println("mode parameter invalid")
			fmt.Fprintf(w, "mode parameter invalid")
			return
		}
	}

	msg, err := t.writeToFile()
	if err != nil {
		log.Println("failed to create obj")
		fmt.Fprintln(w, "failed to create obj :"+msg)
		return
	}

	if len(t.FileNames) == 0 {
		log.Println("missing attachments")
		fmt.Fprintln(w, "missing attachments")
		return
	}

	var commandArgs []string
	var fname string
	fname = t.FileNames[0]
	// fmt.Printf("fname: %s, FileNames : %v, length: %d", fname, t.FileNames, len(t.FileNames))
	for _, fnm := range t.FileNames {
		defer os.Remove(fnm)
	}

	if t.FileFormat == "obj" {
		if !strings.HasSuffix(fname, ".obj") {
			fname = t.FileNames[1]
		}
		log.Println("converting fileformat obj")
		commandArgs = []string{"-i", fname, "-o", "./models/" + strings.TrimSuffix(fname, ".obj") + ".gltf"}
		_, err = exec.Command("obj2gltf", commandArgs...).Output()
		if err != nil {

			log.Println(err)
			fmt.Fprintln(w, "failed to convert to gltf")
			return
		}
		fname = strings.TrimSuffix(fname, ".obj")
	} else if t.FileFormat == "fbx" {
		var msg []byte
		log.Println("converting fileformat fbx")
		commandArgs = []string{"--embed", "-i", fname, "-o", "./models/" + strings.TrimSuffix(fname, ".fbx") + ".gltf"}
		msg, err = exec.Command("./FBX2glTF", commandArgs...).Output()
		if err != nil {
			// log.Fatal(err)
			log.Println(string(msg))
			fmt.Fprintln(w, "failed to convert to gltf")
			return
		}
		fname = strings.TrimSuffix(fname, ".fbx")
	}

	log.Println("convert to gltf successful")

	// convert to usdz
	commandArgs = []string{"./models/" + fname + ".gltf", "./models/" + fname + ".usdz"}
	_, err = exec.Command("usd_from_gltf", commandArgs...).Output()
	if err != nil {
		// log.Fatal(err)
		log.Println(err)
		fmt.Fprint(w, "failed to convert to usdz")
		return
	}
	log.Println("convert to usdz successful")

	fmt.Fprintln(w, fname+".usdz")
	go expireFiles([]string{fname + ".gltf", fname + ".usdz"})
}

func headers(w http.ResponseWriter, req *http.Request) {

	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func main() {
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("ar.portfo.io"), //Your domain here
		Cache:      autocert.DirCache("certs"),             //Folder for storing certificates
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", usdz)
	mux.HandleFunc("/headers", headers)
	mux.Handle("/models/", http.StripPrefix(strings.TrimRight("/models/", "/"), http.FileServer(http.Dir("models"))))
	mainMux := newMiddleware(mux)
	server := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: mainMux,
	}
	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
