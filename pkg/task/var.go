package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// TaskVar wraps a variable
type TaskVar struct {
	Name, Value string
}

func (v *TaskVar) cleanAndCheck() error {
	if strings.Index(v.Name, " ") != -1 {
		return fmt.Errorf("Syntax error near: %s, try remove spaces", v.Name)
	}

	v.Value = strings.TrimLeft(v.Value, " ")
	v.Value = strings.TrimRight(v.Value, " ")

	index1 := strings.Index(v.Value, "\"")
	index2 := strings.LastIndex(v.Value, "\"")

	// remove ""
	if index1 == 0 && index2 == len(v.Value)-1 {
		v.Value = strings.TrimLeft(v.Value, "\"")
		v.Value = strings.TrimRight(v.Value, "\"")
	}
	return nil
}

// NewTaskVar returns a new TaskVar object
// with some smart check such as trim spaces.
// Returns error if there's at least one space in the Name
// or Value is not conform to the right syntax.
func NewTaskVar(name, value string) (TaskVar, error) {
	taskVar := TaskVar{
		Name:  name,
		Value: value,
	}
	err := taskVar.cleanAndCheck()
	return taskVar, err
}

func readVarsFrom(in *bufio.Reader) ([]TaskVar, error) {
	var vars []TaskVar
	var currentVar TaskVar
	var lineCount int
	loop := true
	var strLine string
	var unparsedVar []string
	var line []byte
	var err error

	lineCount = 1
	for loop {
		line, _, err = in.ReadLine()
		if err != nil {
			loop = false
			if err == io.EOF {
				// set to nil because we return this variable
				// and if we have arrived to EOF it's fine
				err = nil
			}
			break
		}
		strLine = string(line)
		strLine = strings.TrimLeft(strLine, " ")

		if strings.Index(strLine, ":") == -1 {
			err = fmt.Errorf("Syntax error at line %d", lineCount)
			loop = false
			break
		}

		unparsedVar = strings.Split(strLine, ":")
		currentVar, err = NewTaskVar(unparsedVar[0], unparsedVar[1])
		if err != nil {
			err = fmt.Errorf("%s at line %d", err.Error(), lineCount)
			loop = false
			break
		} else {
			vars = append(vars, currentVar)
		}
	}

	return vars, err
}

// ReadVars reads variable from the given path.
func ReadVars(path string) ([]TaskVar, error) {
	varsFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	in := bufio.NewReader(varsFile)
	return readVarsFrom(in)
}
