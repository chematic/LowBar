//go:build windows

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func validStyle(v int) bool    { return v >= styleNormal && v <= styleAcrylic }
func validLanguage(v int) bool { return v >= langEnglish && v <= langRussian }

func loadConfig() config {
	cfg := makeConfig()
	file, err := os.Open(configPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logEvent("INFO", "config.load", "configuration file does not exist; defaults used")
		} else {
			logError("config.load", "unable to open configuration", err)
		}
		return cfg
	}
	defer file.Close()

	repaired := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "style":
			v, e := strconv.Atoi(value)
			if e != nil || !validStyle(v) {
				logEvent("ERROR", "config.load", fmt.Sprintf("invalid style=%q; using default=%d", value, cfg.style))
				repaired = true
			} else {
				cfg.style = v
			}
		case "language":
			v, e := strconv.Atoi(value)
			if e != nil || !validLanguage(v) {
				logEvent("ERROR", "config.load", fmt.Sprintf("invalid language=%q; using default=%d", value, cfg.language))
				repaired = true
			} else {
				cfg.language = v
			}
		case "startup":
			cfg.startup = value == "1" || strings.EqualFold(value, "true")
		}
	}
	if err := scanner.Err(); err != nil {
		logError("config.load", "scanner failed", err)
		repaired = true
	}
	if repaired {
		_ = saveConfig(cfg)
	}
	logEvent("INFO", "config.load", fmt.Sprintf("loaded style=%d language=%d startup=%t", cfg.style, cfg.language, cfg.startup))
	return cfg
}

func saveConfig(cfg config) error {
	if !validStyle(cfg.style) {
		cfg.style = styleBlur
	}
	if !validLanguage(cfg.language) {
		cfg.language = langEnglish
	}
	if err := os.MkdirAll(filepath.Dir(configPath()), 0700); err != nil {
		return err
	}
	content := fmt.Sprintf("style=%d\nlanguage=%d\nstartup=%d\n", cfg.style, cfg.language, boolInt(cfg.startup))
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, configPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	logEvent("INFO", "config.save", fmt.Sprintf("saved style=%d language=%d startup=%t path=%s", cfg.style, cfg.language, cfg.startup, configPath()))
	return nil
}
