package frameworks

type Framework interface {
	Name() string
	Detect() bool
	Serve()
}

type DbCredentials struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
}
