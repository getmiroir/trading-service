package subsettings

// Auth represents Auth authentication configuration
type Auth struct {
	LoginURL string `yaml:"login_url"`
}

// DefaultAuth returns Auth with default values
func DefaultAuth() Auth {
	return Auth{
		LoginURL: "https://getmiroir.xyz/login",
	}
}
