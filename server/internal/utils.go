package internal

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ChangeFileNameExtension(fname string, extn string) string {
	base := ExtractFileNameWithoutExtension(fname)
	return fmt.Sprintf("%s.%s", base, extn)
}

func ExtractFileNameWithoutExtension(fname string) string {
	return strings.TrimSuffix(fname, filepath.Ext(fname))
}
