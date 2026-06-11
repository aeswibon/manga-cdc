package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func askOne(prompt, defaultVal string, validate func(string) bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		input = strings.TrimSpace(input)
		if input == "" {
			input = defaultVal
		}
		if validate(input) {
			return input, nil
		}
		fmt.Println("  Invalid choice, try again.")
	}
}
