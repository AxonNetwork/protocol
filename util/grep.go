package util

import (
	"io/ioutil"
	"strings"
)

func GrepExists(file string, str string) (bool, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}
	fileText := string(b)
	return strings.Contains(fileText, str), nil
}
