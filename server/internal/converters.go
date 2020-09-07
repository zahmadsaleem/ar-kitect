package internal

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

func ConvertOBJtoGLTF(fname string, modelsPath string) (res Result) {
	var commandArgs []string

	fname = fmt.Sprintf("%s.%s", ExtractFileNameWithoutExtension(fname), OBJ)

	commandArgs = []string{
		"-i",
		filepath.Join(modelsPath,fname),
		"-o",
		filepath.Join(modelsPath, ChangeFileNameExtension(fname, GLTF)),
	}
	msg, err := exec.Command(OBJtoGLTF, commandArgs...).Output()
	if err != nil {
		res.Message = "failed to convert to GLTF"
		log.Printf("%s %s %s", res.Message, err, string(msg))
		return
	}

	res.Success = true
	res.Message = "successfully converted OBJ to GLTF"
	return
}

func ConvertFBXtoGLTF(fname string, modelsPath string) (res Result) {
	var commandArgs []string
	var msg []byte

	commandArgs = []string{
		"--embed",
		"-i",
		filepath.Join(modelsPath,fname),
		"-o",
		filepath.Join(modelsPath, ChangeFileNameExtension(fname, GLTF)),
	}
	msg, err := exec.Command(FBXtoGLTF, commandArgs...).Output()
	if err != nil {
		log.Printf("FBXtoGLTF error, %s\n", string(msg))
		res.Message = "failed to convert to gltf"
		return
	}
	res.Success = true
	return
}

func ConvertGLTFtoUSDZ(fname string, modelsPath string) (res Result) {
	var commandArgs []string
	commandArgs = []string{
		filepath.Join(modelsPath, fmt.Sprintf("%s.%s", fname, GLTF)),
		filepath.Join(modelsPath, fmt.Sprintf("%s.%s", fname, USDZ)),
	}
	_, err := exec.Command(GLTFtoUSDZ, commandArgs...).Output()
	if err != nil {
		log.Printf("failed to convert to usdz %s\n%s", fname, err)
		res.Message = fmt.Sprintf("failed to convert `%s` to usdz", fname)
		return
	}

	res.Success = true
	res.Message = fname
	return
}
