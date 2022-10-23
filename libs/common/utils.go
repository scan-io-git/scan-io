package common

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/scan-io-git/scan-io/libs/vcs"
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

func ReadReposFile2(inputFile string) ([]vcs.RepositoryParams, error) {
	var wholeFile vcs.ListFuncResult
	var result []vcs.RepositoryParams

	readFile, err := os.Open(inputFile)
	if err != nil {
		return result, err
	}
	defer readFile.Close()

	byteValue, _ := ioutil.ReadAll(readFile)
	err = json.Unmarshal(byteValue, &wholeFile)
	if err != nil {
		return result, err
	}
	return wholeFile.Result, nil
}
