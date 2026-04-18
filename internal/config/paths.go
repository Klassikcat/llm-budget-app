package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	AppDirectoryName       = "llmbudget"
	SettingsFileName       = "settings.json"
	PricesOverrideFileName = "prices.yaml"
	DatabaseFileName       = "llmbudget.sqlite3"
)

type Paths struct {
	ConfigDir          string
	DataDir            string
	SettingsFile       string
	PricesOverrideFile string
	DatabaseFile       string
}

type LookupEnvFunc func(string) (string, bool)

type UserConfigDirFunc func() (string, error)

type PathResolverOptions struct {
	GOOS          string
	LookupEnv     LookupEnvFunc
	UserConfigDir UserConfigDirFunc
}

func ResolvePaths(opts PathResolverOptions) (Paths, error) {
	runtimeOS := opts.GOOS
	if runtimeOS == "" {
		runtimeOS = runtime.GOOS
	}

	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	userConfigDir := opts.UserConfigDir
	if userConfigDir == nil {
		userConfigDir = os.UserConfigDir
	}

	configRoot, err := resolveConfigRoot(runtimeOS, lookupEnv, userConfigDir)
	if err != nil {
		return Paths{}, err
	}

	dataRoot, err := resolveDataRoot(runtimeOS, lookupEnv)
	if err != nil {
		return Paths{}, err
	}

	configDir := filepath.Join(configRoot, AppDirectoryName)
	dataDir := filepath.Join(dataRoot, AppDirectoryName)

	return Paths{
		ConfigDir:          configDir,
		DataDir:            dataDir,
		SettingsFile:       filepath.Join(configDir, SettingsFileName),
		PricesOverrideFile: filepath.Join(configDir, PricesOverrideFileName),
		DatabaseFile:       filepath.Join(dataDir, DatabaseFileName),
	}, nil
}

func resolveConfigRoot(runtimeOS string, lookupEnv LookupEnvFunc, userConfigDir UserConfigDirFunc) (string, error) {
	if dir, err := userConfigDir(); err == nil && strings.TrimSpace(dir) != "" {
		return dir, nil
	}

	switch runtimeOS {
	case "linux":
		if dir, ok := lookupNonEmpty(lookupEnv, "XDG_CONFIG_HOME"); ok {
			return dir, nil
		}

		home, err := requiredHomeDir(runtimeOS, lookupEnv)
		if err != nil {
			return "", err
		}

		return filepath.Join(home, ".config"), nil
	case "darwin":
		home, err := requiredHomeDir(runtimeOS, lookupEnv)
		if err != nil {
			return "", err
		}

		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if dir, ok := lookupNonEmpty(lookupEnv, "APPDATA"); ok {
			return dir, nil
		}

		return "", &SetupError{
			Code:    ErrorCodeConfigDirUnavailable,
			Message: "could not determine the Windows config directory; APPDATA is not set",
		}
	default:
		return "", &SetupError{
			Code:    ErrorCodeConfigDirUnavailable,
			Message: fmt.Sprintf("unsupported operating system %q for config path resolution", runtimeOS),
		}
	}
}

func resolveDataRoot(runtimeOS string, lookupEnv LookupEnvFunc) (string, error) {
	switch runtimeOS {
	case "linux":
		if dir, ok := lookupNonEmpty(lookupEnv, "XDG_DATA_HOME"); ok {
			return dir, nil
		}

		home, err := requiredHomeDir(runtimeOS, lookupEnv)
		if err != nil {
			return "", err
		}

		return filepath.Join(home, ".local", "share"), nil
	case "darwin":
		home, err := requiredHomeDir(runtimeOS, lookupEnv)
		if err != nil {
			return "", err
		}

		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if dir, ok := lookupNonEmpty(lookupEnv, "LOCALAPPDATA"); ok {
			return dir, nil
		}
		if dir, ok := lookupNonEmpty(lookupEnv, "APPDATA"); ok {
			return dir, nil
		}

		return "", &SetupError{
			Code:    ErrorCodeConfigDirUnavailable,
			Message: "could not determine the Windows data directory; LOCALAPPDATA and APPDATA are not set",
		}
	default:
		return "", &SetupError{
			Code:    ErrorCodeConfigDirUnavailable,
			Message: fmt.Sprintf("unsupported operating system %q for data path resolution", runtimeOS),
		}
	}
}

func requiredHomeDir(runtimeOS string, lookupEnv LookupEnvFunc) (string, error) {
	switch runtimeOS {
	case "linux", "darwin":
		if home, ok := lookupNonEmpty(lookupEnv, "HOME"); ok {
			return home, nil
		}
	case "windows":
		if home, ok := lookupNonEmpty(lookupEnv, "USERPROFILE"); ok {
			return home, nil
		}

		drive, driveOK := lookupNonEmpty(lookupEnv, "HOMEDRIVE")
		path, pathOK := lookupNonEmpty(lookupEnv, "HOMEPATH")
		if driveOK && pathOK {
			return filepath.Join(drive, path), nil
		}
	}

	return "", &SetupError{
		Code:    ErrorCodeConfigDirUnavailable,
		Message: "could not determine the user home directory for config path resolution",
		Err:     errors.New("missing home directory environment variables"),
	}
}

func lookupNonEmpty(lookupEnv LookupEnvFunc, key string) (string, bool) {
	if lookupEnv == nil {
		return "", false
	}

	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}

	return value, true
}
