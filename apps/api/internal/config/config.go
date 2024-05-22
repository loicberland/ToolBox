package config

type Config struct {
	Addr       string
	WebOrigins []string
}

func Load() Config {
	return Config{
		Addr:       ":20250",
		WebOrigins: []string{"http://localhost:20251", "http://localhost:3000"},
	}
}
