package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func LoginPasteToken(provider string, r io.Reader) (*AuthCredential, error) {
	fmt.Printf("Paste your API key or session token from %s:\n", providerDisplayName(provider))
	fmt.Print("> ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading token: %w", err)
		}
		return nil, errors.New("no input received")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	return &AuthCredential{
		AccessToken: token,
		Provider:    provider,
		AuthMethod:  "token",
	}, nil
}

func providerDisplayName(provider string) string {
	switch provider {
	case "anthropic":
		return "console.anthropic.com"
	case "openai":
		return "platform.openai.com"
	default:
		return provider
	}
}
