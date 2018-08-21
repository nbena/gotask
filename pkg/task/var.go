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

const (
	// SyntaxError is the prefix of every VarReadingError.
	SyntaxError = "Syntax error at line "

	// VarFileName is the name of the var file.
	VarFileName = ".taskvar"
)

// VarReadingError is thrown when there's an error parsing
// vars file.
type VarReadingError struct {
	Line int
	Desc string
}

func (e VarReadingError) Error() string {
	return fmt.Sprintf(SyntaxError+": %d, %s", e.Line, e.Desc)
}

// Var wraps a variable
type Var struct {
	Name, Value string
}

// ToReplacer returns the variable name in the format
// used for substitution.
func (v *Var) ToReplacer() string {
	return fmt.Sprintf("${%s}", v.Name)
}

func (v *Var) cleanAndCheck() error {

	v.Name = strings.TrimLeft(v.Name, " ")
	v.Name = strings.TrimRight(v.Name, " ")

	if strings.Index(v.Name, " ") != -1 {
		return VarReadingError{
			// Line: line,
			Desc: fmt.Sprintf("near: %s, try remove spaces", v.Name),
		}
		// return fmt.Errorf("Syntax error near: %s, try remove spaces", v.Name)
	}

	v.Value = strings.TrimLeft(v.Value, " ")
	v.Value = strings.TrimRight(v.Value, " ")

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
		// strLine = strings.TrimLeft(strLine, " ")

		if strings.Index(strLine, ":") == -1 {
			err = VarReadingError{
				Line: lineCount,
				Desc: "missing ':'",
			}
			loop = false
			break
		}

		unparsedVar = strings.Split(strLine, ":")
		currentVar, err = NewTaskVar(unparsedVar[0], unparsedVar[1])
		if err != nil {
			loop = false
			break
		} else {
			vars = append(vars, currentVar)
		}
	}

	// type assertion doesn't work inside the loop
	// so call here
	if errMod, ok := err.(VarReadingError); ok {
		errMod.Line = lineCount
		return nil, errMod
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
