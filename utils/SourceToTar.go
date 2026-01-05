package utils

import (
	"fmt"
	"os"
	"strings"
)

func ReadSourcesAsTar(srcPath string) ([]byte, error) {
	// if we are on Windows, inverts the slash
	if IsWindows() {
		srcPath = strings.Replace(srcPath, "/", "\\", -1)
	}

	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("Missing source file")
	}

	var tarFileName string

	if fileInfo.IsDir() || !strings.HasSuffix(srcPath, ".tar") {
		// Creates a temporary dir (cross platform)
		file, err := os.CreateTemp("", "serverledgesource")
		if err != nil {
			return nil, err
		}
		defer func(file *os.File) {
			name := file.Name()
			err := file.Close()
			if err != nil {
				fmt.Printf("Error while closing file '%s': %v", name, err)
				os.Exit(1)
			}
			err = os.Remove(name)
			if err != nil {
				fmt.Printf("Error while trying to remove file '%s': %v", name, err)
				os.Exit(1)
			}
		}(file)

		err = Tar(srcPath, file)
		if err != nil {
			fmt.Printf("Error while trying to tar file '%s'\n", srcPath)
			os.Exit(1)
		}
		tarFileName = file.Name()
	} else {
		// this is already a tar file
		tarFileName = srcPath
	}

	return os.ReadFile(tarFileName)
}
