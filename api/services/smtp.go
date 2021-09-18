package services

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

type Smtp struct {
	Enabled bool
	Path    struct {
		Temp    string
		Forward string
		Aliases string
	}
}

type Techmail struct {
	Enabled bool
	Path    struct {
		Prod string
		Temp string
	}
}

type SmtpString struct {
	Data map[string]interface{}
}

type SmtpSlice struct {
	Data map[string][]string
}

type SmtpMap struct {
	Data map[string]map[string]string
}

var (
	SmtpSettings     Smtp
	TechmailSettings Techmail
)

func smtpCommit(name string, recv string, temp string, forward string, aliases string) error {
	head := temp + "/" + name + ".head"
	conf := forward + "/" + name
	if name == "aliases" {
		conf = aliases + "/" + name
	}
	if err := commit(head, recv, conf); err != nil {
		return err
	}

	return nil
}

func (s *SmtpString) SmtpDownload() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpString) TechMailDownload() error {
	lines := s.Data
	for name, line := range lines {
		recv := TechmailSettings.Path.Temp + "/" + name + ".recv"
		if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
			return err
		}

		head := TechmailSettings.Path.Temp + "/" + name + ".head"
		conf := TechmailSettings.Path.Prod + "/" + name
		if err := commit(head, recv, conf); err != nil {
			return err
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpString) Create() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := add(recv, line.(string)); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpMap) ForwardRename() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		if name == "file_name" {
			for find, replace := range line {
				headOld := temp + "/" + find + ".head"
				headNew := temp + "/" + replace + ".head"
				if err := os.Rename(headOld, headNew); err != nil {
					return err
				}

				recvOld := temp + "/" + find + ".recv"
				recvNew := temp + "/" + replace + ".recv"
				if err := os.Rename(recvOld, recvNew); err != nil {
					return err
				}

				prodFind := forward + "/" + find
				prodReplace := forward + "/" + replace
				if err := os.Rename(prodFind, prodReplace); err != nil {
					return err
				}
			}
		} else {
			recv := temp + "/" + name + ".recv"
			if err := update(recv, line); err != nil {
				return err
			}

			if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
				return err
			}
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpString) ForwardDelete() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		if name == "forward_name" {
			head := temp + "/" + line.(string) + ".head"
			if err := os.Remove(head); err != nil {
				return err
			}

			recv := temp + "/" + line.(string) + ".recv"
			if err := os.Remove(recv); err != nil {
				return err
			}

			prodDelete := forward + "/" + line.(string)
			if err := os.Remove(prodDelete); err != nil {
				return err
			}
		} else {
			recv := temp + "/" + name + ".recv"

			recvData, err := ioutil.ReadFile(recv)
			if err != nil {
				return err
			}
			recvString := string(recvData)
			re := regexp.MustCompile("(?m)^" + line.(string) + `.*$[\r\n]*|[\r\n]+\s+\z`)
			recvString = re.ReplaceAllString(recvString, "")

			if err := ioutil.WriteFile(recv, []byte(recvString), 0644); err != nil {
				return err
			}

			if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
				return err
			}
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpMap) UserUpdate() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := update(recv, line); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func (s *SmtpSlice) UserDelete() error {
	lines := s.Data

	temp := SmtpSettings.Path.Temp
	forward := SmtpSettings.Path.Forward
	aliases := SmtpSettings.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := remove(recv, line); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	if err := newaliases(); err != nil {
		return err
	}

	return nil
}

func newaliases() error {
	cli := exec.Command("/usr/bin/newaliases")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}
