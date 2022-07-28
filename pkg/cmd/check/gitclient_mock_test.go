package check_test

import (
	"fmt"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
)

type mockedCommand struct {
	dir                  string
	expectedStringReturn string
	expectedErrorReturn  error
	args                 []string
}

type mockGitClient struct {
	mockedCommands []mockedCommand
}

func NewMockGitClient() gitclient.Interface {
	gc := mockGitClient{}
	return &gc
}

func (m *mockGitClient) Command(dir string, args ...string) (string, error) {
	for _, mockedCommand := range m.mockedCommands {
		if dir == mockedCommand.dir && isArrayEqual(args, mockedCommand.args) {
			return mockedCommand.expectedStringReturn, mockedCommand.expectedErrorReturn
		}
	}
	return "", fmt.Errorf("command not mocked with Dir %s and args %v", dir, args)
}

func isArrayEqual(args1 []string, args2 []string) bool {
	if len(args1) != len(args2) {
		return false
	}

	for index, arg1 := range args1 {
		if arg1 != args2[index] {
			return false
		}
	}

	return true
}

func (m *mockGitClient) addMockedCommand(dir string, expectedStringReturn string, expectedErrorReturn error, args ...string) {
	if m.mockedCommands == nil {
		m.mockedCommands = []mockedCommand{}
	}

	m.mockedCommands = append(m.mockedCommands, mockedCommand{
		dir:                  dir,
		expectedStringReturn: expectedStringReturn,
		expectedErrorReturn:  expectedErrorReturn,
		args:                 args,
	})
}

func (m *mockGitClient) reset() {
	m.mockedCommands = nil
}
