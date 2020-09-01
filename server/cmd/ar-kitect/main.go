package main

import (
	"ar-kitect/server/internal"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func pathsMustExist(paths ...string) {
	for _, p := range paths {
		abspath, _ := filepath.Abs(p)
		if _, err := os.Stat(p); os.IsNotExist(err) || p == "" {
			panic(fmt.Sprintf("path '%s' is empty or not accessible", abspath))
		}
		log.Println(p)
	}
}

func main() {
	port := fmt.Sprintf(":%s", os.Getenv(internal.ServerPortEnvVar))
	staticPath, _ := os.LookupEnv(internal.AppStaticPathEnvVar)
	modelsPath, _ := os.LookupEnv(internal.ModelsPathEnvVar)

	pathsMustExist(staticPath, modelsPath)
	log.Printf("static path %s, models path %s", staticPath, modelsPath)

	server := internal.CreateServer(modelsPath, staticPath, port)

	log.Printf("starting server on port %s", port)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
