package utils

import (
	"fmt"

	"github.com/pterm/pterm"
)

func Success(msg string, args ...any) {
	pterm.Success.Printf(msg+"\n", args...)
}

func Info(msg string, args ...any) {
	pterm.Info.Printf(msg+"\n", args...)
}

func Warn(msg string, args ...any) {
	pterm.Warning.Printf(msg+"\n", args...)
}

func Error(msg string, args ...any) {
	pterm.Error.Printf(msg+"\n", args...)
}

func Debug(msg string, args ...any) {
	pterm.Debug.Printf(msg+"\n", args...)
}

func Box(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	pterm.DefaultBox.Println(formatted)
}

func Fatal(err error) {
	if axErr, ok := err.(*AxenError); ok {
		pterm.Error.Printf("%s: %s\n", axErr.Code, axErr.Message)
	} else {
		pterm.Error.Println(err.Error())
	}
}

var StartSpinner = func(text string) (*pterm.SpinnerPrinter, error) {
	return pterm.DefaultSpinner.Start(text)
}

func init() {
	// Override the default blocky spinner with a modern dots spinner
	pterm.DefaultSpinner.Sequence = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}
