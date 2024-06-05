package strategies

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/getlantern/algeneva"
)

const (
	testStrategiesFile = "test_strategies.txt"
)

var (
	strategy    *algeneva.HTTPStrategy
	strategyStr string
)

func init() {
	strat, err := loadStrategy(testStrategiesFile)
	if err != nil {
		panic(err)
	}

	strategy, err = algeneva.NewHTTPStrategy(strat)
	if err != nil {
		panic(err)
	}

	strategyStr = strat
}

func loadStrategy(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read test strategies file: %w", err)
	}

	lines := bytes.Split(content, []byte("\n"))
	for _, strat := range lines {
		strat = bytes.TrimSpace(strat)
		if len(strat) == 0 {
			break
		}

		if strat[0] != '=' {
			return string(strat), nil
		}
	}

	return "", errors.New("no strategies found")
}

func GetStrategy() *algeneva.HTTPStrategy {
	return strategy
}

func WriteResult(pass bool) error {
	content, err := os.ReadFile(testStrategiesFile)
	if err != nil {
		return fmt.Errorf("failed to read test strategies file: %w", err)
	}

	lines := bytes.Split(content, []byte("\n"))
	for i, strat := range lines {
		strat = bytes.TrimSpace(strat)
		if len(strat) == 0 {
			break
		}

		if strat[0] == '=' {
			strat = strat[7:] // Skip the "=PASS= " or "=FAIL= "
		}

		if bytes.Equal(strat, []byte(strategyStr)) {
			if pass {
				lines[i] = []byte("=PASS= " + strategyStr)
			} else {
				lines[i] = []byte("=FAIL= " + strategyStr)
			}

			content = bytes.Join(lines, []byte("\n"))
			return os.WriteFile(testStrategiesFile, content, 0644)
		}
	}

	return errors.New("strategy not found")
}
