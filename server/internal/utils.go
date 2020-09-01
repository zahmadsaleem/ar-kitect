package internal

import (
	"fmt"
	"strings"
)

func ChangeFileNameExtension(fname string, extn string) string {
	split := strings.Split(fname, ".")
	joined := strings.Join(split[:len(split)-1], ".")
	return fmt.Sprintf("%s.%s", joined, extn)
}

func ExtractFileNameWithoutExtension(fname string) string {
	split := strings.Split(fname, ".")
	return strings.Join(split[:len(split)-1], ".")
}

