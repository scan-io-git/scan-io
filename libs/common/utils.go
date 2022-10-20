package common

import (
	"bufio"
	"os"
)

func ReadReposFile(inputFile string) ([]string, error) {
	readFile, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	lines := []string{}
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	return lines, nil
}
