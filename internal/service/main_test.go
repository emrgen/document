package service

import (
	"github.com/emrgen/tinydoc/internal/tester"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tester.Setup()
	code := m.Run()

	os.Exit(code)
}
