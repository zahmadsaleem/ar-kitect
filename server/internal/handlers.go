package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func convertHandler(modelsPath string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
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

		res = t.receiveFiles(req, modelsPath)
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
			defer os.Remove(filepath.Join(modelsPath, fnm))
		}

		if t.FileFormat == OBJ {
			log.Println("converting OBJ to GLTF")
			res = ConvertOBJtoGLTF(fname, modelsPath)
			if !res.Success {
				return
			}
		} else if t.FileFormat == FBX {
			log.Println("converting FBX to GLTF")
			res = ConvertFBXtoGLTF(fname, modelsPath)
			if !res.Success {
				return
			}
		}

		log.Println("convert to GLTF successful")
		fname = ExtractFileNameWithoutExtension(fname)

		res = ConvertGLTFtoUSDZ(fname, modelsPath)
		if !res.Success {
			return
		}

		log.Println("convert to USDZ successful")
	}
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

func incomingHeadersHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("headers requested")
	for name, headers := range req.Header {
		for _, h := range headers {
			_, _ = fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}
