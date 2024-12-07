package service

import (
	"github.com/emrgen/document/internal/tester"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tester.Setup()
	code := m.Run()

	os.Exit(code)
}
