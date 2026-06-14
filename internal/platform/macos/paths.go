package macos

import (
	"os"
	"path/filepath"
)

func (Platform) DefaultConfigDir(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, appName), nil
}

func (Platform) DefaultLogDir(appName string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, appName, "logs"), nil
}
