package internal

import (
	"testing"
)

func TestChangeFileNameExtension(t *testing.T) {
	initial := "xyz-a.dsds.gltf"
	expected := "xyz-a.dsds.usdz"
	if got := ChangeFileNameExtension(initial, USDZ); expected != got {
		t.Errorf("expected %s got %s", expected, got)
	}
}

func TestExtractFileNameWithoutExtension(t *testing.T) {
	initial := "xyz-a.dsds.gltf"
	expected := "xyz-a.dsds"
	if got := ExtractFileNameWithoutExtension(initial); expected != got {
		t.Errorf("expected %s got %s", expected, got)
	}
}

