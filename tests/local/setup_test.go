package local

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	godotenv.Load(".env.test")
	code := m.Run()
	os.Exit(code)
}
