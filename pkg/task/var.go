// go-task, a simple client-server task runner
// Copyright (C) 2018 nbena
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Var wraps a variable
type Var struct {
	Name, Value string
}

func (v *Var) cleanAndCheck() error {
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
func NewTaskVar(name, value string) (Var, error) {
	taskVar := Var{
		Name:  name,
		Value: value,
	}
	err := taskVar.cleanAndCheck()
	return taskVar, err
}

func readVarsFrom(in *bufio.Reader) ([]Var, error) {
	var vars []Var
	var currentVar Var
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
func ReadVars(path string) ([]Var, error) {
	varsFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	in := bufio.NewReader(varsFile)
	return readVarsFrom(in)
}
