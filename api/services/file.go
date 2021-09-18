package services

import (
	"io"
	"io/ioutil"
	"os"
	"regexp"
)

func beginTransaction(recv string, backup string) error {
	recvFile, err := os.OpenFile(recv, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer recvFile.Close()
	backupFile, err := os.OpenFile(backup, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer backupFile.Close()

	if _, err := io.Copy(backupFile, recvFile); err != nil {
		return err
	}
	if err := backupFile.Sync(); err != nil {
		return err
	}

	return nil
}

func rollback(recv string, backup string) error {
	recvFile, err := os.OpenFile(recv, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer recvFile.Close()
	backupFile, err := os.OpenFile(backup, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer backupFile.Close()

	if _, err := io.Copy(recvFile, backupFile); err != nil {
		return err
	}
	if err := recvFile.Sync(); err != nil {
		return err
	}

	return nil
}

func commit(head string, recv string, conf string) error {
	if _, err := os.OpenFile(head, os.O_RDONLY|os.O_CREATE, 0644); err != nil {
		return err
	}

	headData, err := ioutil.ReadFile(head)
	if err != nil {
		return err
	}
	recvData, err := ioutil.ReadFile(recv)
	if err != nil {
		return err
	}
	confFile, err := os.OpenFile(conf, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer confFile.Close()

	if _, err := confFile.WriteString(string(headData)); err != nil {
		return err
	}
	if _, err := confFile.WriteString(string(recvData)); err != nil {
		return err
	}

	return nil
}

func add(recv string, data string) error {
	recvFile, err := os.OpenFile(recv, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer recvFile.Close()

	if _, err := recvFile.WriteString(data); err != nil {
		return err
	}

	return nil
}

func update(recv string, data map[string]string) error {
	recvData, err := ioutil.ReadFile(recv)
	if err != nil {
		return err
	}

	recvString := string(recvData)
	for find, replace := range data {
		re := regexp.MustCompile("(?m)^" + find + `.*$[\r\n]*|[\r\n]+\s+\z`)
		recvString = re.ReplaceAllString(recvString, replace)
	}

	if err := ioutil.WriteFile(recv, []byte(recvString), 0644); err != nil {
		return err
	}

	return nil
}

func remove(recv string, data []string) error {
	recvData, err := ioutil.ReadFile(recv)
	if err != nil {
		return err
	}

	recvString := string(recvData)
	for _, line := range data {
		re := regexp.MustCompile("(?m)^" + line + `.*$[\r\n]*|[\r\n]+\s+\z`)
		recvString = re.ReplaceAllString(recvString, "")
	}

	if err := ioutil.WriteFile(recv, []byte(recvString), 0644); err != nil {
		return err
	}

	return nil
}
