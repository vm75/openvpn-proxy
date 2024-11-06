package utils

import "os"

func UpdateContent(content string, file string) (bool, error) {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return true, nil
	}
	if string(fileContent) == content {
		return false, nil
	}
	err = os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}
