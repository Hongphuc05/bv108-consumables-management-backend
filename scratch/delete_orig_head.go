package main

import (
	"fmt"
	"os"
)

func main() {
	files := []string{
		".git/ORIG_HEAD",
		".git/FETCH_HEAD",
	}

	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			err = os.Remove(f)
			if err != nil {
				fmt.Printf("Error deleting %s: %v\n", f, err)
			} else {
				fmt.Printf("Successfully deleted corrupted transient file: %s\n", f)
			}
		} else {
			fmt.Printf("File %s does not exist or cannot be accessed.\n", f)
		}
	}
}
