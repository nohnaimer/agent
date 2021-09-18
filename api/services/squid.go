package services

import (
	"errors"
	"io/ioutil"
	"os/exec"
)

type Squid struct {
	Enabled bool
	Path    struct {
		Prod string
		Temp string
	}
}

type SquidString struct {
	Data map[string]interface{}
}

var SquidSettings Squid

func (s *SquidString) Download() error {
	lines := s.Data
	for name, line := range lines {
		recv := SquidSettings.Path.Temp + "/" + name + ".recv"
		if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
			return err
		}

		head := SquidSettings.Path.Temp + "/" + name + ".head"
		conf := SquidSettings.Path.Prod + "/" + name
		if err := commit(head, recv, conf); err != nil {
			return err
		}
	}

	if err := restartSquid(); err != nil {
		return err
	}

	return nil
}

func restartSquid() error {
	cli := exec.Command("/usr/sbin/squid", "-k", "reconfigure")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}
