package config_test

import (
	"os"
	"pg-bash-exporter/internal/config"
	"testing"
)

// tests that loading a good config works
func TestLoadSuccess(t *testing.T) {
	good_yaml := `
server:
  listen_address: ":1234"
`
	// make a temp file
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal("couldnt make temp file")
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file %s: %v", f.Name(), err)
		}
	}() // clean up

	if _, err := f.Write([]byte(good_yaml)); err != nil {
		t.Fatal("couldnt write to temp file")
	}
	f.Close()

	var cfg config.Config
	err = config.Load(f.Name(), &cfg)

	if err != nil {
		t.Error("load failed but should have passed")
	}

	if cfg.Server.ListenAddress != ":1234" {
		t.Error("config data not loaded right")
	}
}

// tests that a bad path fails
func TestLoadBadPath(t *testing.T) {
	var cfg config.Config
	err := config.Load("/tmp/this/file/does/not/exist.yaml", &cfg)
	if err == nil {
		t.Error("load passed but should have failed")
	}
}

// tests that bad yaml fails
func TestLoadBadYAML(t *testing.T) {
	bad_yaml := "server: 'bad"

	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal("couldnt make temp file")
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file %s: %v", f.Name(), err)
		}
	}()

	if _, err := f.Write([]byte(bad_yaml)); err != nil {
		t.Fatal("couldnt write to temp file")
	}
	f.Close()

	var cfg config.Config
	err = config.Load(f.Name(), &cfg)
	if err == nil {
		t.Error("load passed with bad yaml but should have failed")
	}
}
