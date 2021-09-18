package services

import (
	"errors"
	"io/ioutil"
	"os/exec"
)

type Samba struct {
	Enabled bool
	Path    struct {
		Prod string
		Temp string
	}
}

type ShareString struct {
	Data map[string]interface{}
}

var ShareSettings Samba

func (s *ShareString) Download() error {
	line := s.Data["samba"]

	return download(line)
}

func download(line interface{}) error {
	temp := ShareSettings.Path.Temp
	recv := temp + "/smb.conf.recv"
	if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
		return err
	}

	head := temp + "/smb.conf.head"
	conf := temp + "/smb.conf"
	if err := commit(head, recv, conf); err != nil {
		return err
	}

	//if err := sambaTestConfig(conf); err != nil {
	//	return err
	//}

	confData, err := ioutil.ReadFile(conf)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(ShareSettings.Path.Prod+"/smb.conf", confData, 0644); err != nil {
		return err
	}

	if err = restartSamba(); err != nil {
		return err
	}

	return nil
}

func (s *ShareString) Create() error {
	zfs := int(s.Data["is_zfs"].(float64))
	quota := s.Data["quota"]
	path := s.Data["path"]
	zfsPath := s.Data["zfs_path"]
	if zfs == 1 {
		cli := exec.Command("/sbin/zfs", "create", "-o", "refquota="+quota.(string), zfsPath.(string))
		output, err := cli.CombinedOutput()
		if err != nil {
			return errors.New(err.Error() + ": " + string(output))
		}
	} else {
		cli := exec.Command("/usr/bin/mkdir", "-p", path.(string))
		output, err := cli.CombinedOutput()
		if err != nil {
			return errors.New(err.Error() + ": " + string(output))
		}
	}

	cli := exec.Command("/usr/bin/chmod", "777", path.(string))
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	line := s.Data["samba"]

	temp := ShareSettings.Path.Temp
	recv := temp + "/smb.conf.recv"
	backup := temp + "/smb.conf.recv.backup"
	if err := beginTransaction(recv, backup); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	if err := add(recv, line.(string)); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	head := temp + "/smb.conf.head"
	conf := temp + "/smb.conf"
	if err := commit(head, recv, conf); err != nil {
		return err
	}

	//if err := sambaTestConfig(conf); err != nil {
	//	return err
	//}

	confData, err := ioutil.ReadFile(conf)
	if err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	if err := ioutil.WriteFile(ShareSettings.Path.Prod+"/smb.conf", confData, 0644); err != nil {
		return err
	}

	if err = restartSamba(); err != nil {
		return err
	}

	return nil
}

func (s *ShareString) Quota() error {
	quota := s.Data["quota"]
	zfsPath := s.Data["zfs_path"]

	cli := exec.Command("/sbin/zfs", "set", "refquota="+quota.(string), zfsPath.(string))
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}

func (s *ShareString) Delete() error {
	zfsPath := s.Data["zfs_path"]
	backupServer := s.Data["backup_server"]
	backupServerPool := s.Data["backup_server_pool"]
	name := s.Data["name"]

	cli := exec.Command("/sbin/zfs", "list", zfsPath.(string))
	_, err := cli.CombinedOutput()
	if err != nil {
		return nil
	}

	cli = exec.Command("/sbin/zfs", "destroy", "-r", zfsPath.(string))
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	cli = exec.Command("ssh", backupServer.(string), "zfs", "list", backupServerPool.(string)+"/"+name.(string))
	_, err = cli.CombinedOutput()
	if err != nil {
		return nil
	}

	cli = exec.Command("ssh", backupServer.(string), "zfs", "destroy", "-r", backupServerPool.(string)+"/"+name.(string))
	output, err = cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	line := s.Data["samba"]

	return download(line)
}

//func sambaTestConfig(conf string) error {
//	cli := exec.Command("/usr/bin/testparm", "-s", conf)
//	output, err := cli.CombinedOutput()
//	if err != nil {
//		return errors.New(err.Error() + ": " + string(output))
//	}
//
//	return nil
//}

func restartSamba() error {
	cli := exec.Command("/usr/bin/systemctl", "restart", "smb.service")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}
