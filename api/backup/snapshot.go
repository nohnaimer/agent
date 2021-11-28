package backup

import (
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"os"
	"os/exec"
	"time"
)

const (
	RemoteBackupDisable   = 1
	RotatePeriodDay       = "d"
	RotatePeriodDayCount  = -1
	RotatePeriodWeek      = "w"
	RotatePeriodWeekCount = -7
	RotateWeekDay         = 1
)

type SnapshotMap struct {
	Data []snapshot `json:"data"`
}

type snapshot struct {
	Name             string `json:"name"`
	Path             string `json:"path"`
	ZfsPath          string `json:"zfs_path"`
	BackupServer     string `json:"backup_server"`
	BackupServerPool string `json:"backup_server_pool"`
	RotationPeriod   int    `json:"rotation_period"`
	RotationType     string `json:"rotation_type"`
	IsRemoteBackup   int    `json:"is_remote_backup"`
}

var (
	now             time.Time
	previous        = 0
	rotation        = 0
	currentSnapshot string
)

func (s *SnapshotMap) Backup() error {
	now = time.Now()
	shares := s.Data

	for _, share := range shares {
		if share.isFsNotExist() {
			continue
		}

		if share.RotationType == RotatePeriodWeek && int(now.Weekday()) != RotateWeekDay {
			continue
		}

		share.init()

		err := share.create()
		if err != nil {
			continue
		}

		share.createBackupLink()

		_ = share.clone()

		_ = share.rotate()
	}

	return nil
}

func (s *snapshot) isFsNotExist() bool {
	cli := exec.Command("/sbin/zfs", "list", s.ZfsPath)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - File system doesn't exists: " + s.Name)
		sentry.CaptureException(message)
		return true
	}

	return false
}

func (s *snapshot) init() {
	currentSnapshot = s.ZfsPath + "@" + now.Format("2006-01-02")

	switch s.RotationType {
	case RotatePeriodDay:
		previous = RotatePeriodDayCount
		rotation = RotatePeriodDayCount * s.RotationPeriod
	case RotatePeriodWeek:
		previous = RotatePeriodWeekCount
		rotation = RotatePeriodWeekCount * s.RotationPeriod
	}
}

func (s *snapshot) create() error {
	if s.isSnapshotExist(currentSnapshot) {
		return nil
	}

	cli := exec.Command("/sbin/zfs", "snapshot", currentSnapshot)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - Error create snapshot: " + s.Name)
		sentry.CaptureException(message)
		return message
	}

	return nil
}

func (s *snapshot) createBackupLink() {
	if _, err := os.Stat(s.Path + "/___backups___"); os.IsNotExist(err) {
		err = os.Chdir(s.Path)
		if err != nil {
			message := errors.New(err.Error() + ". Error createBackupLink: " + s.Name)
			sentry.CaptureException(message)
		} else {
			cli := exec.Command("/usr/bin/ln", "-s", ".zfs/snapshot", "___backups___")
			_, _ = cli.CombinedOutput()
		}
	}
}

func (s *snapshot) clone() error {
	if s.IsRemoteBackup == RemoteBackupDisable {
		return nil
	}

	return s.send()
}

func (s *snapshot) send() error {
	if _, err := os.Stat(s.Path + "/.zfs/snapshot/" + currentSnapshot); os.IsNotExist(err) {
		return nil
	}

	if s.isRemoteFsExist() {
		previousDate := now.AddDate(0, 0, previous)
		previousSnapshot := s.ZfsPath + "@" + previousDate.Format("2006-01-02")

		if _, err := os.Stat(s.Path + "/.zfs/snapshot/" + previousDate.Format("2006-01-02")); !os.IsNotExist(err) {
			command := fmt.Sprintf("zfs send -i %s %s | ssh %s zfs recv -F %s/%s", previousSnapshot, currentSnapshot, s.BackupServer, s.BackupServerPool, s.Name)
			cli := exec.Command("/usr/bin/bash", "-c", command)
			output, err := cli.CombinedOutput()
			if err != nil {
				message := errors.New(err.Error() + ": " + string(output) + " - Error send increment snapshot to remote server")
				sentry.CaptureException(message)
				return message
			}

			return nil
		}
	}

	if s.isRemoteFsExist() {
		s.destroyRemoteFs()
	}

	err := s.createRemoteFs()
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	command := fmt.Sprintf("zfs send %s | ssh %s zfs recv -F %s/%s", currentSnapshot, s.BackupServer, s.BackupServerPool, s.Name)
	cli := exec.Command("/usr/bin/bash", "-c", command)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - Error send snapshot to remote server")
		sentry.CaptureException(message)
		return message
	}

	return nil
}

func (s *snapshot) isSnapshotExist(snapshot string) bool {
	cli := exec.Command("/sbin/zfs", "list", snapshot)
	_, err := cli.CombinedOutput()

	return err == nil
}

func (s *snapshot) isRemoteFsExist() bool {
	cli := exec.Command("ssh", s.BackupServer, "zfs", "list", s.BackupServerPool+"/"+s.Name)
	_, err := cli.CombinedOutput()
	return err == nil
}

func (s *snapshot) isRemoteSnapshotExist(snapshot string) bool {
	cli := exec.Command("ssh", s.BackupServer, "zfs", "list", snapshot)
	_, err := cli.CombinedOutput()
	return err == nil
}

func (s *snapshot) createRemoteFs() error {
	cli := exec.Command("ssh", s.BackupServer, "zfs", "create", "-o", "compression=on", s.BackupServerPool+"/"+s.Name)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - Error create remote file system")
		sentry.CaptureException(message)
		return message
	}

	return nil
}

func (s *snapshot) destroyRemoteFs() {
	cli := exec.Command("ssh", s.BackupServer, "zfs", "destroy", "-fr", s.BackupServerPool+"/"+s.Name)
	_, _ = cli.CombinedOutput()
}

func (s *snapshot) rotate() error {
	rotationDate := now.AddDate(0, 0, rotation)
	rotationSnapshot := s.ZfsPath + "@" + rotationDate.Format("2006-01-02")
	if s.isSnapshotExist(rotationSnapshot) {
		cli := exec.Command("/sbin/zfs", "destroy", "-fr", rotationSnapshot)
		output, err := cli.CombinedOutput()
		if err != nil {
			message := errors.New(err.Error() + ": " + string(output) + " - Error rotate snapshot: " + rotationSnapshot)
			sentry.CaptureException(message)
			return message
		}
	}

	if s.IsRemoteBackup == RemoteBackupDisable {
		return nil
	}

	rotationRemoteSnapshot := s.BackupServerPool + "/" + s.Name + "@" + rotationDate.Format("2006-01-02")
	if s.isRemoteSnapshotExist(rotationRemoteSnapshot) {
		cli := exec.Command("ssh", s.BackupServer, "zfs", "destroy", "-fr", rotationRemoteSnapshot)
		output, err := cli.CombinedOutput()
		if err != nil {
			message := errors.New(err.Error() + ": " + string(output) + " - Error rotate remote snapshot: " + rotationRemoteSnapshot)
			sentry.CaptureException(message)
			return message
		}
	}

	return nil
}
