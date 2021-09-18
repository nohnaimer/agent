package services

import (
	"errors"
	"io/ioutil"
	"os/exec"
)

type Dhcp struct {
	Enabled bool
	Path    struct {
		Prod string
		Temp string
	}
}

type DhcpString struct {
	Data map[string]interface{}
}

type DhcpSlice struct {
	Data map[string][]string
}

type DhcpMap struct {
	Data map[string]map[string]string
}

var DhcpSettings Dhcp

func (d *DhcpString) Download() error {
	return crud(d.Data["dhcpd"], true)
}

func (d *DhcpString) Create() error {
	return crud(d.Data["dhcpd"], false)
}

func (d *DhcpMap) Update() error {
	return crud(d.Data["dhcpd"], false)
}

func (d *DhcpSlice) Delete() error {
	return crud(d.Data["dhcpd"], false)
}

func crud(line interface{}, download bool) error {
	temp := DhcpSettings.Path.Temp
	head := temp + "/dhcpd.conf.head"
	recv := temp + "/dhcpd.conf.recv"
	backup := temp + "/dhcpd.conf.recv.backup"
	conf := temp + "/dhcpd.conf"

	if err := beginTransaction(recv, backup); err != nil {
		if err = rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	switch line.(type) {
	case string:
		if download {
			if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
				if err = rollback(recv, backup); err != nil {
					return err
				}
				return err
			}
		} else {
			if err := add(recv, line.(string)); err != nil {
				if err = rollback(recv, backup); err != nil {
					return err
				}
				return err
			}
		}
	case map[string]string:
		if err := update(recv, line.(map[string]string)); err != nil {
			if err = rollback(recv, backup); err != nil {
				return err
			}
			return err
		}
	case []string:
		if err := remove(recv, line.([]string)); err != nil {
			if err = rollback(recv, backup); err != nil {
				return err
			}
			return err
		}
	}

	if err := commit(head, recv, conf); err != nil {
		if err = rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	//if err := testConfig(tempConf); err != nil {
	//	if err := Rollback(recv, backup); err != nil {
	//		return err
	//	}
	//	return err
	//}

	data, err := ioutil.ReadFile(conf)
	if err != nil {
		if err = rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	conf = DhcpSettings.Path.Prod + "/dhcpd.conf"
	if err = ioutil.WriteFile(conf, data, 0644); err != nil {
		if err = rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	if err = restartDhcp(); err != nil {
		if err = rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	return nil
}

//func testConfig(conf string) error {
//	cli := exec.Command("/usr/sbin/dhcpd", "-t", "-cf", conf)
//	output, err := cli.CombinedOutput()
//	if err != nil {
//		return errors.New(err.Error() + ": " + string(output))
//	}
//
//	return nil
//}

func restartDhcp() error {
	cli := exec.Command("/usr/bin/systemctl", "restart", "dhcpd.service")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}
