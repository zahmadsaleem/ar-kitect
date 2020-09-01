package internal

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

func ConvertOBJtoGLTF(fname string) (res Result) {
	var commandArgs []string

	fname = fmt.Sprintf("%s.%s", filepath.Base(fname), OBJ)

	commandArgs = []string{
		"-i",
		fname,
		"-o",
		fmt.Sprintf("./models/%s", ChangeFileNameExtension(fname, GLTF)),
	}
	_, err := exec.Command(OBJtoGLTF, commandArgs...).Output()
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

	commandArgs = []string{
		"--embed",
		"-i",
		fname,
		"-o",
		fmt.Sprintf("./models/%s", ChangeFileNameExtension(fname, GLTF)),
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

func ConvertGLTFtoUSDZ(fname string) (res Result) {
	var commandArgs []string
	commandArgs = []string{
		fmt.Sprintf("./models/%s.%s", fname, GLTF),
		fmt.Sprintf("./models/%s.%s", fname, USDZ),
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
