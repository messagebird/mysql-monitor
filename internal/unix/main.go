package unix

import (
	"bytes"
	"github.com/pkg/errors"
	"io/ioutil"
	"os/exec"
)

func GetProcessList() (*bytes.Buffer, error) {
	cmd := exec.Command("ps", "aux")

	return execute(cmd)
}

func GetTop() (*bytes.Buffer, error) {
	cmd := exec.Command("top", "-b", "-n 1")

	return execute(cmd)
}

func execute(cmd *exec.Cmd) (*bytes.Buffer, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not pipe stdout")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "execution of command to get ps list failed")
	}

	b, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, errors.Wrap(err, "could not read command's output")
	}

	buff := bytes.NewBuffer(b)

	if err := cmd.Wait(); err != nil {
		return nil, errors.Wrap(err, "waiting for command to finish failed")
	}

	return buff, nil
}
