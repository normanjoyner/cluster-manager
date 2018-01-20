package kubectl

import (
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
)

// Kubectl is used to keep track of commands ran through the
// kubectl cli
type Kubectl struct {
	Command *exec.Cmd
	Input   []byte
	Error   []byte
	Output  []byte
}

var kubectl = env.KubectlPath()

// NewApplyCmd streams the input to the kubectl apply command
func NewApplyCmd(path string) *Kubectl {
	return &Kubectl{
		Command: exec.Command(kubectl, "apply", "-f", path),
		Input:   nil,
		Error:   make([]byte, 0),
		Output:  make([]byte, 0),
	}
}

// NewDeleteCmd stremes the input to the kubectl delete command
func NewDeleteCmd(path string) *Kubectl {
	return &Kubectl{
		Command: exec.Command(kubectl, "delete", "-f", path),
		Input:   nil,
		Error:   make([]byte, 0),
		Output:  make([]byte, 0),
	}
}

// NewDeleteByLabelCmd deletes all in namespaces that match labels. Which are
// 'and' together
func NewDeleteByLabelCmd(namespace string, labels ...string) *Kubectl {
	return &Kubectl{
		Command: exec.Command(kubectl, "delete", "all", "--namespace", namespace, "-l", strings.Join(labels, ",")),
		Input:   nil,
		Error:   make([]byte, 0),
		Output:  make([]byte, 0),
	}
}

// Run runs the kubectl command created
func (kc *Kubectl) Run() error {
	cmd := kc.Command
	log.Debug("Running command: ", cmd.Path, cmd.Args)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	inputErr := kc.writeInput(stdin, kc.Input)
	kc.Output, _ = ioutil.ReadAll(stdout)
	kc.Error, _ = ioutil.ReadAll(stderr)
	err = cmd.Wait()

	switch {
	case inputErr != nil && err != nil:
		return fmt.Errorf("Error with input: %s, Error with command: %s", inputErr.Error(), err.Error())
	case inputErr != nil:
		return inputErr
	case err != nil:
		return err
	}

	return nil
}

func (kc *Kubectl) writeInput(stdin io.WriteCloser, input []byte) error {
	if input == nil {
		return nil
	}

	defer stdin.Close()
	_, err := stdin.Write(input)

	return err
}
