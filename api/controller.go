package api

import (
	"agent/api/backup"
	"agent/api/services"
	"github.com/go-ozzo/ozzo-routing/v2"
)

func actionDhcpDownload(c *routing.Context) error {
	var dhcp services.DhcpString
	if err := c.Read(&dhcp); err != nil {
		return err
	}

	return dhcp.Download()
}

func actionDhcpCreate(c *routing.Context) error {
	var dhcp services.DhcpString
	if err := c.Read(&dhcp); err != nil {
		return err
	}

	return dhcp.Create()
}

func actionDhcpUpdate(c *routing.Context) error {
	var dhcp services.DhcpMap
	if err := c.Read(&dhcp); err != nil {
		return err
	}

	return dhcp.Update()
}

func actionDhcpDelete(c *routing.Context) error {
	var dhcp services.DhcpSlice
	if err := c.Read(&dhcp); err != nil {
		return err
	}

	return dhcp.Delete()
}

func actionSmtpDownload(c *routing.Context) error {
	var smtp services.SmtpString
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.SmtpDownload()
}

func actionSmtpCreate(c *routing.Context) error {
	var smtp services.SmtpString
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.Create()
}

func actionSmtpForwardRename(c *routing.Context) error {
	var smtp services.SmtpMap
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.ForwardRename()
}

func actionSmtpForwardDelete(c *routing.Context) error {
	var smtp services.SmtpString
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.ForwardDelete()
}

func actionSmtpUserUpdate(c *routing.Context) error {
	var smtp services.SmtpMap
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.UserUpdate()
}

func actionSmtpUserDelete(c *routing.Context) error {
	var smtp services.SmtpSlice
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.UserDelete()
}

func actionTechMailDownload(c *routing.Context) error {
	var smtp services.SmtpString
	if err := c.Read(&smtp); err != nil {
		return err
	}

	return smtp.TechMailDownload()
}

func actionSquidDownload(c *routing.Context) error {
	var squid services.SquidString
	if err := c.Read(&squid); err != nil {
		return err
	}

	return squid.Download()
}

func actionSambaDownload(c *routing.Context) error {
	var samba services.ShareString
	if err := c.Read(&samba); err != nil {
		return err
	}

	return samba.Download()
}

func actionSambaCreate(c *routing.Context) error {
	var samba services.ShareString
	if err := c.Read(&samba); err != nil {
		return err
	}

	return samba.Create()
}

func actionSambaQuota(c *routing.Context) error {
	var samba services.ShareString
	if err := c.Read(&samba); err != nil {
		return err
	}

	return samba.Quota()
}

func actionSambaDelete(c *routing.Context) error {
	var samba services.ShareString
	if err := c.Read(&samba); err != nil {
		return err
	}

	return samba.Delete()
}

func actionSambaBackup(c *routing.Context) error {
	var samba backup.SnapshotMap
	if err := c.Read(&samba); err != nil {
		return err
	}

	return samba.Backup()
}
