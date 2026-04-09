package config

type Config struct {
	TicketPattern string
}

func Default() Config {
	return Config{
		TicketPattern: `\b[A-Z][A-Z0-9]+-\d+\b`,
	}
}
