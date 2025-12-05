package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadConfigFile_MissingFileReturnsError(t *testing.T) {
    wd, err := os.Getwd()
    if err != nil {
        t.Fatalf("getwd: %v", err)
    }
    t.Cleanup(func() {
        _ = os.Chdir(wd)
    })

    tmp := t.TempDir()
    if err := os.Chdir(tmp); err != nil {
        t.Fatalf("chdir: %v", err)
    }

    if _, err := loadConfigFile(); err == nil {
        t.Fatalf("expected error for missing config.json")
    }
}

func TestLoadConfigFile_ReadsExistingFile(t *testing.T) {
    wd, err := os.Getwd()
    if err != nil {
        t.Fatalf("getwd: %v", err)
    }
    t.Cleanup(func() {
        _ = os.Chdir(wd)
    })

    tmp := t.TempDir()
    if err := os.Chdir(tmp); err != nil {
        t.Fatalf("chdir: %v", err)
    }

    cfg := ConfigFile{APIServerPort: 1234, UseDefaultCoins: true}
    content := `{"api_server_port":1234,"use_default_coins":true}`
    path := filepath.Join(tmp, "config.json")
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatalf("write config.json: %v", err)
    }

    got, err := loadConfigFile()
    if err != nil {
        t.Fatalf("loadConfigFile returned error: %v", err)
    }
    if got == nil {
        t.Fatalf("expected config, got nil")
    }
    if got.APIServerPort != cfg.APIServerPort {
        t.Fatalf("api_server_port mismatch: want %d, got %d", cfg.APIServerPort, got.APIServerPort)
    }
    if got.UseDefaultCoins != cfg.UseDefaultCoins {
        t.Fatalf("use_default_coins mismatch: want %v, got %v", cfg.UseDefaultCoins, got.UseDefaultCoins)
    }
}
