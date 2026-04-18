package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolvePathsLinuxUsesXDGConfigHome(t *testing.T) {
	env := mapEnv(
		"XDG_CONFIG_HOME", "/tmp/xdg-config",
		"XDG_DATA_HOME", "/tmp/xdg-data",
		"HOME", "/tmp/home",
	)

	paths, err := ResolvePaths(PathResolverOptions{
		GOOS:      "linux",
		LookupEnv: env.LookupEnv,
		UserConfigDir: func() (string, error) {
			return "", errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if got, want := paths.ConfigDir, filepath.Join("/tmp/xdg-config", AppDirectoryName); got != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := paths.DataDir, filepath.Join("/tmp/xdg-data", AppDirectoryName); got != want {
		t.Fatalf("DataDir = %q, want %q", got, want)
	}
	if got, want := paths.SettingsFile, filepath.Join("/tmp/xdg-config", AppDirectoryName, SettingsFileName); got != want {
		t.Fatalf("SettingsFile = %q, want %q", got, want)
	}
}

func TestResolvePathsLinuxFallsBackToHomeConvention(t *testing.T) {
	env := mapEnv("HOME", "/users/tester")

	paths, err := ResolvePaths(PathResolverOptions{
		GOOS:      "linux",
		LookupEnv: env.LookupEnv,
		UserConfigDir: func() (string, error) {
			return "", errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if got, want := paths.ConfigDir, filepath.Join("/users/tester", ".config", AppDirectoryName); got != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := paths.DataDir, filepath.Join("/users/tester", ".local", "share", AppDirectoryName); got != want {
		t.Fatalf("DataDir = %q, want %q", got, want)
	}
}

func TestResolvePathsDarwinFallback(t *testing.T) {
	env := mapEnv("HOME", "/Users/tester")

	paths, err := ResolvePaths(PathResolverOptions{
		GOOS:      "darwin",
		LookupEnv: env.LookupEnv,
		UserConfigDir: func() (string, error) {
			return "", errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	wantRoot := filepath.Join("/Users/tester", "Library", "Application Support", AppDirectoryName)
	if paths.ConfigDir != wantRoot {
		t.Fatalf("ConfigDir = %q, want %q", paths.ConfigDir, wantRoot)
	}
	if paths.DataDir != wantRoot {
		t.Fatalf("DataDir = %q, want %q", paths.DataDir, wantRoot)
	}
}

func TestResolvePathsWindowsFallback(t *testing.T) {
	env := mapEnv(
		"APPDATA", `C:\Users\tester\AppData\Roaming`,
		"LOCALAPPDATA", `C:\Users\tester\AppData\Local`,
	)

	paths, err := ResolvePaths(PathResolverOptions{
		GOOS:      "windows",
		LookupEnv: env.LookupEnv,
		UserConfigDir: func() (string, error) {
			return "", errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if got, want := paths.ConfigDir, filepath.Join(`C:\Users\tester\AppData\Roaming`, AppDirectoryName); got != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := paths.DataDir, filepath.Join(`C:\Users\tester\AppData\Local`, AppDirectoryName); got != want {
		t.Fatalf("DataDir = %q, want %q", got, want)
	}
	if got, want := paths.DatabaseFile, filepath.Join(`C:\Users\tester\AppData\Local`, AppDirectoryName, DatabaseFileName); got != want {
		t.Fatalf("DatabaseFile = %q, want %q", got, want)
	}
}

func TestResolvePathsErrorsWithoutHomeFallback(t *testing.T) {
	_, err := ResolvePaths(PathResolverOptions{
		GOOS:      "linux",
		LookupEnv: mapEnv().LookupEnv,
		UserConfigDir: func() (string, error) {
			return "", errors.New("boom")
		},
	})
	if err == nil {
		t.Fatal("ResolvePaths() error = nil, want error")
	}
	if !IsSetupErrorCode(err, ErrorCodeConfigDirUnavailable) {
		t.Fatalf("ResolvePaths() error = %v, want config_dir_unavailable", err)
	}
}

type envMap map[string]string

func mapEnv(keyValues ...string) envMap {
	env := envMap{}
	for i := 0; i+1 < len(keyValues); i += 2 {
		env[keyValues[i]] = keyValues[i+1]
	}

	return env
}

func (e envMap) LookupEnv(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}
