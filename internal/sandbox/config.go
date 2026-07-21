package sandbox

type Config struct {
	RootDir       string
	AllowedDirs   []string
	AllowedHosts  []string
	BlockedHosts  []string
	MaxMemoryMB   int
	MaxCPUSeconds int
	TimeoutSecs   int
	Command       string
	Args          []string
}

func DefaultConfig() Config {
	return Config{
		MaxMemoryMB:   512,
		MaxCPUSeconds: 300,
		TimeoutSecs:   0,
	}
}
