// printBuildInfo выводит информацию о сборке.
package utils

import "fmt"

// Глобальные переменные для сборки
var (
	BuildVersion string
	BuildDate    string
	BuildCommit  string
)

// PrintBuildInfo выводит информацию о сборке.
func PrintBuildInfo() {
	version := "N/A"
	if BuildVersion != "" {
		version = BuildVersion
	}

	date := "N/A"
	if BuildDate != "" {
		date = BuildDate
	}

	commit := "N/A"
	if BuildCommit != "" {
		commit = BuildCommit
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
