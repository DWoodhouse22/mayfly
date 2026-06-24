package cli

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func PromptForInput(label string, reader *bufio.Reader, validateFunc validate) string {
	for {
		fmt.Printf("%s: ", label)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if validateFunc != nil {
			if err := validateFunc(input); err != nil {
				fmt.Println(err)
				continue
			}
		}
		return input
	}
}

type validate func(input string) error

func ValidateIP(input string) error {
	ip := net.ParseIP(input)
	if ip == nil {
		return fmt.Errorf("invalid ip string")
	}
	return nil
}

func ValidateSSHUser(input string) error {
	if input == "" {
		return fmt.Errorf("user cannot be empty")
	}

	if strings.Contains(input, " ") {
		return fmt.Errorf("user cannot contain spaces")
	}

	if strings.ContainsAny(input, ":@") {
		return fmt.Errorf("invalid characters in user")
	}
	return nil
}
