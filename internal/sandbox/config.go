package sandbox

type Config struct {
	RootDir      string
	AllowedDirs  []string
	BlockedDirs  []string
	AllowedHosts []string
	BlockedHosts []string
	MaxMemoryMB  int
	MaxCPUSeconds int
	Timeout      int
}

func DefaultConfig() Config {
	return Config{
		RootDir:      "/tmp/vault-root",
		AllowedDirs:  []string{"/usr", "/bin", "/lib", "/etc"},
		BlockedDirs:  []string{"/home", "/root", "/tmp/vault-audit.db"},
		AllowedHosts: []string{},
		BlockedHosts: []string{},
		MaxMemoryMB:  512,
		MaxCPUSeconds: 300,
		Timeout:      3600,
	}
}
