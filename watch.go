package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/fatih/color"
)

type OutputType struct {
	Type string `json:"type"`
}

type OutputError struct {
	Title   string        `json:"title"`
	Path    string        `json:"path"`
	Message []interface{} `json:"message"`
}

type OutputErrors struct {
	Errors []struct {
		Path     string `json:"path"`
		Problems []struct {
			Title  string `json:"title"`
			Region struct {
				Start struct {
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"start"`
			} `json:"region"`
			Message []interface{} `json:"message"`
		} `json:"problems"`
	} `json:"errors"`
}

var (
	currentDirectory  string
	pathToElmCompiler *string
	pathToMainFile    *string
)

func init() {
	log.SetFlags(0)
	color.NoColor = false

	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	pathToElmCompiler = flag.String("elm", "/usr/local/bin/elm", "path to elm compiler")
	pathToMainFile = flag.String("main", "./src/elm/Main.elm", "path to main file")
	flag.Parse()
}

func main() {
	runElmMake()
}

func runElmMake() {
	cmd := exec.Command(*pathToElmCompiler, "make", *pathToMainFile, "--output=/dev/null", "--report=json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		fmt.Println("It Works!")
		return
	}

	var outputType OutputType
	if err := json.Unmarshal(stderr.Bytes(), &outputType); err != nil {
		log.Fatalln(err, stderr.String())
	}

	switch outputType.Type {
	case "error":
		var elmErr OutputError
		err := json.Unmarshal(stderr.Bytes(), &elmErr)
		if err != nil {
			log.Fatalln(err, stderr.String())
		}
		printError(elmErr)
	case "compile-errors":
		var elmErr OutputErrors
		err := json.Unmarshal(stderr.Bytes(), &elmErr)
		if err != nil {
			log.Fatalln(err, stderr.String())
		}
		printErrors(elmErr)
	default:
		fmt.Println(stderr.String())
	}

	os.Exit(1)
}

func printError(output OutputError) {
	printHeader(output.Title, output.Path, 1, 0)
	printMessage(getMessage(output.Message), 0)
}

func printErrors(output OutputErrors) {
	for _, e := range output.Errors {
		for _, p := range e.Problems {
			fmt.Println()
			printHeader(p.Title, e.Path, p.Region.Start.Line, p.Region.Start.Column)
			printMessage(getMessage(p.Message), p.Region.Start.Line)
		}
	}
}

func printHeader(title, relativePath string, line, column int) {
	c := color.New(color.FgCyan)
	_, err := c.Print(title, " -- ", path.Join(currentDirectory, relativePath), ":"+strconv.Itoa(line)+":"+strconv.Itoa(column), "\n")
	if err != nil {
		log.Fatal(err)
	}
}

func getMessage(output []interface{}) string {
	var message string
	for _, m := range output {
		switch v := m.(type) {
		case string:
			message += v
		case map[string]interface{}:
			message += getStyledMessagePart(v)
		default:
			fmt.Println("Problem with go parser, unrecognized type", reflect.TypeOf(m))
		}
	}
	return message
}

func getStyledMessagePart(v map[string]interface{}) string {
	// get formatting values
	s := v["string"].(string)
	isBold := v["bold"].(bool)
	isUnderline := v["underline"].(bool)
	// handle nil color
	var msgColor string
	switch col := v["color"].(type) {
	case string:
		msgColor = col
	}

	c := color.New()
	switch strings.ToLower(msgColor) {
	case "red":
		c.Add(color.FgRed)
	case "yellow":
		c.Add(color.FgYellow)
	case "green":
		c.Add(color.FgGreen)
	}
	if isBold {
		c.Add(color.Bold)
	}
	if isUnderline {
		c.Add(color.Underline)
	}

	return c.Sprint(s)
}

func printMessage(message string, errorLineNumber int) {
	split := strings.Split(message, "\n")

	var resultMessage []string
	for _, s := range split {
		testable := strings.Trim(s, " \t")
		if len(testable) > 0 && unicode.IsNumber(rune(testable[0])) {
			if strings.HasPrefix(testable, strconv.Itoa(errorLineNumber)) || strings.HasPrefix(testable, strconv.Itoa(errorLineNumber-1)) {
				resultMessage = append(resultMessage, s)
			}
		} else if testable != "" {
			resultMessage = append(resultMessage, s)
		}
	}
	fmt.Println(strings.Join(resultMessage, "\n"))
}
