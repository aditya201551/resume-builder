package jobmap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	JSearchKey      string
	JSearchProvider string
	JSearchLimit    int
	HimalayasKey    string
}

func LoadConfig() (Config, error) {
	c := Config{JSearchKey: os.Getenv("JSEARCH_API_KEY"), HimalayasKey: os.Getenv("HIMALAYAS_API_KEY"), JSearchLimit: 200}
	c.JSearchProvider = strings.ToLower(strings.TrimSpace(os.Getenv("JSEARCH_PROVIDER")))
	if c.JSearchProvider == "" {
		c.JSearchProvider = "openwebninja"
	}
	if c.JSearchProvider != "openwebninja" && c.JSearchProvider != "rapidapi" {
		return c, fmt.Errorf("JSEARCH_PROVIDER must be openwebninja or rapidapi")
	}
	if raw := os.Getenv("JSEARCH_MONTHLY_LIMIT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 200 {
			return c, fmt.Errorf("JSEARCH_MONTHLY_LIMIT must be between 0 and the prototype request cap of 200")
		}
		c.JSearchLimit = n
	}
	return c, nil
}
