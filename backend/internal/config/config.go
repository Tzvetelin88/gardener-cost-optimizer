package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIAddress               string
	DataSourceMode           string
	GardenerKubeconfig       string
	GardenerContext          string
	PrometheusURL            string
	ShootKubeconfigMap       map[string]string
	RefreshInterval          time.Duration
	EnableFallbackData       bool
	FrontendOrigin           string
	IdleThresholdPercent     float64
	TargetUtilizationPercent float64
	ActionLogPath            string
}

func Load() Config {
	refresh := 2 * time.Minute
	if raw := os.Getenv("REFRESH_INTERVAL_SECONDS"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			refresh = time.Duration(seconds) * time.Second
		}
	}

	return Config{
		APIAddress:               env("API_ADDR", ":8080"),
		DataSourceMode:           dataSourceMode(),
		GardenerKubeconfig:       os.Getenv("GARDENER_KUBECONFIG"),
		GardenerContext:          os.Getenv("GARDENER_CONTEXT"),
		PrometheusURL:            os.Getenv("PROMETHEUS_URL"),
		ShootKubeconfigMap:       parseMap(os.Getenv("SHOOT_KUBECONFIG_MAP")),
		RefreshInterval:          refresh,
		EnableFallbackData:       env("ENABLE_FALLBACK_DATA", "true") == "true",
		FrontendOrigin:           env("FRONTEND_ORIGIN", "*"),
		IdleThresholdPercent:     parseFloat(os.Getenv("IDLE_THRESHOLD"), 75),
		TargetUtilizationPercent: parseFloat(os.Getenv("TARGET_UTILIZATION"), 70),
		ActionLogPath:            env("ACTION_LOG_PATH", "./data/actions.jsonl"),
	}
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func parseMap(raw string) map[string]string {
	values := map[string]string{}
	if raw == "" {
		return values
	}

	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}

		values[parts[0]] = parts[1]
	}

	return values
}

func parseFloat(raw string, fallback float64) float64 {
	if raw != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			return v
		}
	}

	return fallback
}

func dataSourceMode() string {
	if mode := strings.ToLower(strings.TrimSpace(os.Getenv("DATA_SOURCE"))); mode != "" {
		switch mode {
		case "mock", "real", "auto":
			return mode
		}
	}

	if env("ENABLE_FALLBACK_DATA", "true") == "true" {
		return "auto"
	}

	return "real"
}
