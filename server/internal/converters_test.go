package internal

import "testing"

func TestConvertGLTFtoUSDZ(t *testing.T) {
	// obj2gltf must be installed
	res := ConvertOBJtoGLTF("Intermediate", "assets")
	if !res.Success {
		t.Error(res.Message)
	}
}
