package utils

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тест с установленной информацией о сборке
func TestPrintBuildInfoFull(t *testing.T) {
	// Подменяем глобальные переменные сборки
	originalVersion := BuildVersion
	originalDate := BuildDate
	originalCommit := BuildCommit

	BuildVersion = "v1.2.3"
	BuildDate = "2023-01-01T00:00:00Z"
	BuildCommit = "abc123def456"

	defer func() {
		// Восстанавливаем значения
		BuildVersion = originalVersion
		BuildDate = originalDate
		BuildCommit = originalCommit
	}()

	// Перехватываем stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintBuildInfo()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	expected := "Build version: v1.2.3\nBuild date: 2023-01-01T00:00:00Z\nBuild commit: abc123def456\n"
	assert.Equal(t, expected, string(out))
}

// Тест, когда информация о сборке не установлена (все N/A)
func TestPrintBuildInfoNA(t *testing.T) {
	// Подменяем глобальные переменные сборки
	originalVersion := BuildVersion
	originalDate := BuildDate
	originalCommit := BuildCommit

	BuildVersion = ""
	BuildDate = ""
	BuildCommit = ""

	defer func() {
		// Восстанавливаем значения
		BuildVersion = originalVersion
		BuildDate = originalDate
		BuildCommit = originalCommit
	}()

	// Перехватываем stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintBuildInfo()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	expected := "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n"
	assert.Equal(t, expected, string(out))
}

// Тест смешанного сценария (некоторые поля установлены, некоторые нет)
func TestPrintBuildInfoMixed(t *testing.T) {
	// Подменяем глобальные переменные сборки
	originalVersion := BuildVersion
	originalDate := BuildDate
	originalCommit := BuildCommit

	BuildVersion = "v2.0.0"
	BuildDate = ""
	BuildCommit = "xyz789"

	defer func() {
		// Восстанавливаем значения
		BuildVersion = originalVersion
		BuildDate = originalDate
		BuildCommit = originalCommit
	}()

	// Перехватываем stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintBuildInfo()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	expected := "Build version: v2.0.0\nBuild date: N/A\nBuild commit: xyz789\n"
	assert.Equal(t, expected, string(out))
}
