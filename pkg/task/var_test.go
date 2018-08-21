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
	"os"
	"reflect"
	"strings"
	"testing"
)

type varStrTestCase struct {
	input         []string
	expected      []Var
	withError     bool
	expectedError VarReadingError
}

type varFileTestCase struct {
	varStrTestCase
	file string
}

func (v *varStrTestCase) manageResult(result []Var, err error, t *testing.T) {
	if v.withError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else {
			if v.expectedError.Error() != err.Error() {
				t.Errorf("Error mismatch:\ngot: %s\nexpected: %s",
					err.Error(), v.expectedError.Error())
			}
		}
	} else {
		if err != nil {
			t.Errorf("Got error while expecting none: %s\n", err.Error())
		} else if !reflect.DeepEqual(result, v.expected) {
			t.Errorf("Mismatch result:\ngot: %v\nexpected: %v",
				result, v.expected)
		}
	}
}

func (v *varStrTestCase) doTest(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(strings.Join(v.input, "\n")))

	result, err := readVarsFrom(reader)
	v.manageResult(result, err, t)
}

func (v *varFileTestCase) doTest(t *testing.T) {
	file, err := os.Create(v.file)
	if err != nil {
		t.Fatalf("Cannot create file: %s", err.Error())
	}
	for _, variable := range v.varStrTestCase.input {
		if _, err = fmt.Fprintf(file, "%s\n", variable); err != nil {
			t.Errorf("Cannot write to file: %s", err.Error())
		}
	}
	if file.Close() != nil {
		t.Errorf("Fail to close the file: %s", err.Error())
	}

	result, err := ReadVars(v.file)
	v.manageResult(result, err, t)

	os.Remove(file.Name())
}

var inputOk = varStrTestCase{
	input: []string{
		"output: /dev/null",
		"input: /dev/stdin",
		"tmp: /tmp",
		"nic: eth0",
		" go: pher",
	},
	expected: []Var{
		{
			Name:  "output",
			Value: "/dev/null",
		}, {
			Name:  "input",
			Value: "/dev/stdin",
		}, {
			Name:  "tmp",
			Value: "/tmp",
		}, {
			Name:  "nic",
			Value: "eth0",
		}, {
			Name:  "go",
			Value: "pher",
		},
	},
	withError: false,
}

var inputNameSpaceError = varStrTestCase{
	input: []string{
		"go p: her",
	},
	expected:  []Var{},
	withError: true,
	expectedError: VarReadingError{
		Desc: "near: go p, try remove spaces",
		Line: 1,
	},
}

var strAllInputs = []varStrTestCase{
	inputOk,
	inputNameSpaceError,
}

var fileAllInputs = []varFileTestCase{
	{
		varStrTestCase: inputOk,
		file:           "input_ok.vars",
	}, {
		varStrTestCase: inputNameSpaceError,
		file:           "input_not_ok.vars",
	},
}

func TestVarsFromStr(t *testing.T) {
	for _, testCase := range strAllInputs {
		testCase.doTest(t)
	}
}

func TestVarsFromFile(t *testing.T) {
	for _, testCase := range fileAllInputs {
		testCase.doTest(t)
	}
}
