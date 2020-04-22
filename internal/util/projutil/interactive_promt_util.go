// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package projutil

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// InteractiveLevel captures the user preference on the generation of interactive
// commands.
type InteractiveLevel int

const (
	// User has not turned interactive mode on or off, default to off.
	InteractiveSoftOff InteractiveLevel = iota
	// User has explicitly turned interactive mode off.
	InteractiveHardOff
	// User only explicitly turned interactive mode on.
	InteractiveOnAll
)

func printMessage(msg string, isOptional bool) {
	fmt.Println()
	if isOptional {
		fmt.Print(strings.TrimSpace(msg) + " (optional): " + "\n" + "> ")
	} else {
		fmt.Print(strings.TrimSpace(msg) + " (required): " + "\n" + "> ")
	}
}

func GetRequiredInput(msg string) string {
	return getRequiredInput(os.Stdin, msg)
}

func getRequiredInput(rd io.Reader, msg string) string {
	reader := bufio.NewReader(rd)

	for {
		printMessage(msg, false)
		value := readInput(reader)
		if value != "" {
			return value
		}
		fmt.Printf("Input is required. ")
	}
}

func GetOptionalInput(msg string) string {
	printMessage(msg, true)
	value := readInput(bufio.NewReader(os.Stdin))
	return value
}

func GetStringArray(msg string) []string {
	return getStringArray(os.Stdin, msg)
}

func getStringArray(rd io.Reader, msg string) []string {
	reader := bufio.NewReader(rd)
	for {
		printMessage(msg, false)
		value := readArray(reader)
		if len(value) != 0 && len(value[0]) != 0 {
			return value
		}
		fmt.Printf("No list provided. ")
	}
}

// readstdin reads a line from stdin and returns the value.
func readLine(reader *bufio.Reader) string {
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error when reading input: %v", err)
	}
	return strings.TrimSpace(text)
}

func readInput(reader *bufio.Reader) string {
	for {
		text := readLine(reader)
		return text
	}
}

// readArray parses the line from stdin, returns an array
// of words.
func readArray(reader *bufio.Reader) []string {
	arr := make([]string, 0)
	text := readLine(reader)

	for _, words := range strings.Split(text, ",") {
		arr = append(arr, strings.TrimSpace(words))
	}
	return arr
}
