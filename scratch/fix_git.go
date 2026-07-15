package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	logPath := ".git/logs/refs/heads/main"
	refPath := ".git/refs/heads/main"

	logFile, err := os.Open(logPath)
	if err != nil {
		fmt.Printf("Error opening git log file: %v\n", err)
		return
	}
	defer logFile.Close()

	var lastLine string
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading git log file: %v\n", err)
		return
	}

	if lastLine == "" {
		fmt.Println("Git reflog file is empty.")
		return
	}

	fmt.Printf("Last reflog line: %s\n", lastLine)
	parts := strings.Split(lastLine, " ")
	if len(parts) < 2 {
		fmt.Println("Invalid reflog format.")
		return
	}

	// The second part is the new SHA-1 commit hash
	newSHA := parts[1]
	if len(newSHA) != 40 {
		fmt.Printf("Extracted hash %q is not 40 characters long.\n", newSHA)
		return
	}

	fmt.Printf("Found latest valid commit hash: %s\n", newSHA)

	// Write it to .git/refs/heads/main
	err = os.WriteFile(refPath, []byte(newSHA+"\n"), 0644)
	if err != nil {
		fmt.Printf("Error writing to %s: %v\n", refPath, err)
		return
	}

	fmt.Println("SUCCESS: Corrected .git/refs/heads/main. Your Git repository has been repaired!")
}
