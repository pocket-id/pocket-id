package common

import (
	"fmt"
	"os"
	"strings"
)

// FileEnvVarSuffix is the suffix of the environment variable that contains the path to the file with the value of another environment variable
// For example, the value of "DB_CONNECTION_STRING" can be loaded from the file whose path is in "DB_CONNECTION_STRING_FILE"
const FileEnvVarSuffix = "_FILE"

// LoadEnvVarFromFile loads the value of the environment variable envVarName from the file referenced by the corresponding "_FILE" environment variable
// The second return value is false when the "_FILE" variable is not set, in which case the caller should use the value of the environment variable itself
// The content of the file is returned as-is: use LoadStringEnvVarFromFile for values that are used as strings
func LoadEnvVarFromFile(envVarName string) ([]byte, bool, error) {
	fileEnvVarName := envVarName + FileEnvVarSuffix
	fileName := os.Getenv(fileEnvVarName)
	if fileName == "" {
		return nil, false, nil
	}

	// #nosec G703 - Path is passed by the admin
	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file '%s' for env var %s: %w", fileName, fileEnvVarName, err)
	}

	return fileContent, true, nil
}

// LoadStringEnvVarFromFile is like LoadEnvVarFromFile, but returns the content of the file as a string, with leading and trailing whitespace removed
// Trimming is required because tools that write secrets to a file, including shell redirections and most editors, normally append a trailing newline
func LoadStringEnvVarFromFile(envVarName string) (string, bool, error) {
	fileContent, ok, err := LoadEnvVarFromFile(envVarName)
	if err != nil || !ok {
		return "", false, err
	}

	return strings.TrimSpace(string(fileContent)), true, nil
}
