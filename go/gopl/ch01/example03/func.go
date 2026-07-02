package example03

import (
	"strings"
)

func ForJoin(args []string) string {
	var s, sep string
	for i := 0; i < len(args); i++ {
		s += sep + args[i]
		sep = " "
	}
	return s
}

func StringsJoin(args []string) string {
	return strings.Join(args, " ")
}
