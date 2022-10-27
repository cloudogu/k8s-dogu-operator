package util

import (
	"fmt"
	"strings"
)

// GetMapKeysAsString returns the key of a map as a string in form: "key1, key2, key3".
func GetMapKeysAsString(input map[string]string) string {
	output := ""

	for key := range input {
		output = fmt.Sprintf("%s, %s", output, key)
	}

	return strings.TrimLeft(output, ", ")
}
