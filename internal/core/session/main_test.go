package session

import (
	"os"
	"testing"

	"github.com/aki/amux/internal/test"
)

func TestMain(m *testing.M) {
	test.InitTestLogger()
	os.Exit(m.Run())
}
