package backup

import (
	"errors"
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
	Data map[string]map[int]snapshot
}

type snapshot struct {
	Name             string
	Path             string
	ZfsPath          string
	BackupServer     string
	BackupServerPool string
	RotationPeriod   int
	RotationType     string
	IsRemoteBackup   int
}

var (
	now             = time.Now()
	previous        = 0
	rotation        = 0
	currentSnapshot string
)

func (s *SnapshotMap) Backup() error {
	shares := s.Data["data"]

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
		return false
	}

	return true
}

func (s *snapshot) init() {
	currentSnapshot = s.ZfsPath + "@" + now.Format("2006-02-01")

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
		cli := exec.Command("/usr/bin/ln", "-s", s.Path+"/.zfs/snapshot", s.Path+"/___backups___")
		_, _ = cli.CombinedOutput()
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

	previousDate := now.AddDate(0, 0, previous)
	previousSnapshot := s.ZfsPath + "@" + previousDate.Format("2006-02-01")

	if _, err := os.Stat(s.Path + "/.zfs/snapshot/" + previousSnapshot); !os.IsNotExist(err) {
		if s.isRemoteSnapshotExist() {
			cli := exec.Command("/sbin/zfs", "send", "-i", previousSnapshot, currentSnapshot, "|", "ssh", s.BackupServer, "zfs", "recv", "-F", s.BackupServerPool+"/"+s.Name)
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

	cli := exec.Command("/sbin/zfs", "send", currentSnapshot, "|", "ssh", s.BackupServer, "zfs", "recv", "-F", s.BackupServerPool+"/"+s.Name)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - Error send snapshot to remote server")
		sentry.CaptureException(message)
		return message
	}

	return nil
}

func (s *snapshot) isRemoteFsExist() bool {
	cli := exec.Command("ssh", s.BackupServer, "zfs", "list", s.BackupServerPool+"/"+s.Name)
	_, err := cli.CombinedOutput()
	return err == nil
}

func (s *snapshot) isRemoteSnapshotExist() bool {
	previousDate := now.AddDate(0, 0, previous)
	previousRemoteSnapshot := s.BackupServerPool + "/" + s.Name + "@" + previousDate.Format("2006-02-01")
	cli := exec.Command("ssh", s.BackupServer, "zfs", "list", previousRemoteSnapshot)
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
	cli := exec.Command("ssh", s.BackupServer, "zfs", "destroy", "-r", s.BackupServerPool+"/"+s.Name)
	_, _ = cli.CombinedOutput()
}

func (s *snapshot) rotate() error {
	rotationDate := now.AddDate(0, 0, rotation)
	rotationSnapshot := s.ZfsPath + "@" + rotationDate.Format("2006-02-01")
	cli := exec.Command("/sbin/zfs", "destroy", rotationSnapshot)
	output, err := cli.CombinedOutput()
	if err != nil {
		message := errors.New(err.Error() + ": " + string(output) + " - Error rotate snapshot: " + rotationSnapshot)
		sentry.CaptureException(message)
		return message
	}

	if s.IsRemoteBackup == RemoteBackupDisable {
		return nil
	}

	if s.isRemoteSnapshotExist() {
		rotationRemoteSnapshot := s.BackupServerPool + "/" + s.Name + "@" + rotationDate.Format("2006-02-01")
		cli = exec.Command("ssh", s.BackupServer, "zfs", "destroy", rotationRemoteSnapshot)
		output, err = cli.CombinedOutput()
		if err != nil {
			message := errors.New(err.Error() + ": " + string(output) + " - Error rotate remote snapshot: " + rotationRemoteSnapshot)
			sentry.CaptureException(message)
			return message
		}
	}

	return nil
}
