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
	resultsFile        = "results.txt"
)

var strategy *algeneva.HTTPStrategy

func init() {
	strat, err := readStrategy(testStrategiesFile)
	if err != nil {
		panic(err)
	}

	strategy, err = algeneva.NewHTTPStrategy(strat)
	if err != nil {
		panic(err)
	}
}

func readStrategy(filename string) (string, error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open test strategies file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}

	if n == 0 {
		return "", errors.New("no strategies found in file")
	}

	strat, _, _ := bytes.Cut(buf, []byte("\n"))
	strat = bytes.TrimSpace(strat)

	return string(strat), nil
}

func GetStrategy() *algeneva.HTTPStrategy {
	return strategy
}

func WriteResult(msg string) (int, error) {
	f, err := os.OpenFile(resultsFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open results file: %w", err)
	}
	defer f.Close()

	n, err := f.WriteString("[" + strategy.String() + "] " + msg + "\n")
	if err != nil {
		return 0, fmt.Errorf("failed to write to results file: %w", err)
	}

	return n, deleteStrategy()
}

func deleteStrategy() error {
	content, err := os.ReadFile(testStrategiesFile)
	if err != nil {
		return fmt.Errorf("failed to read test strategies file: %w", err)
	}

	nlIdx := bytes.IndexByte(content, '\n')
	if nlIdx == -1 {
		return nil
	}

	return os.WriteFile(testStrategiesFile, content[nlIdx+1:], 0644)
}
