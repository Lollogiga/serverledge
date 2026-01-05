package utils

import "os"

func IsWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}
