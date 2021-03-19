package main

import (
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/go-ozzo/ozzo-routing/v2"
	"github.com/go-ozzo/ozzo-routing/v2/access"
	"github.com/go-ozzo/ozzo-routing/v2/content"
	"github.com/go-ozzo/ozzo-routing/v2/fault"
	"github.com/go-ozzo/ozzo-routing/v2/file"
	"github.com/go-ozzo/ozzo-routing/v2/slash"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

type Config struct {
	Port   string
	Sentry struct {
		Dsn string
	}
	Dhcp struct {
		Enabled bool
		Path    struct {
			Prod string
			Temp string
		}
	}
	Smtp struct {
		Enabled bool
		Path    struct {
			Temp    string
			Forward string
			Aliases string
		}
	}
	Squid struct {
		Enabled bool
		Path    struct {
			Prod string
			Temp string
		}
	}
	Techmail struct {
		Enabled bool
		Path    struct {
			Prod string
			Temp string
		}
	}
	Samba struct {
		Enabled bool
		Path    struct {
			Prod string
			Temp string
		}
	}
}

type dataString struct {
	Data map[string]interface{}
}

type dataSlice struct {
	Data map[string][]string
}

type dataMap struct {
	Data map[string]map[string]string
}

type Response struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

var (
	cfg Config
)

func readConfig(cfg *Config) error {
	confFile, err := os.Open("agent.yml")
	if err != nil {
		return err
	}
	defer confFile.Close()

	decoder := yaml.NewDecoder(confFile)
	err = decoder.Decode(cfg)
	if err != nil {
		return err
	}
	err = envconfig.Process("", cfg)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := readConfig(&cfg)
	if err != nil {
		log.Fatalf("Config.read: %s", err)
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn: cfg.Sentry.Dsn,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	router := routing.New()

	router.Use(
		// all these handlers are shared by every route
		access.Logger(log.Printf),
		slash.Remover(http.StatusMovedPermanently),
		fault.Recovery(log.Printf),
	)

	v0 := router.Group("/v0")
	v0.Use(
		// these handlers are shared by the routes in the api group only
		content.TypeNegotiator(content.JSON),
	)

	if cfg.Dhcp.Enabled {
		dhcp := v0.Group("/dhcp")

		dhcp.Post("/config/download", func(c *routing.Context) error {
			if err := actionDhcpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success dhcp download!"})
		})

		network := dhcp.Group("/network")
		network.Post("/create", func(c *routing.Context) error {
			if err := actionDhcpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success create dhcp network!"})
		})
		network.Put("/update", func(c *routing.Context) error {
			if err := actionDhcpUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success update dhcp network!"})
		})
		network.Delete("/delete", func(c *routing.Context) error {
			if err := actionDhcpDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success delete dhcp network!"})
		})

		host := dhcp.Group("/host")
		host.Post("/create", func(c *routing.Context) error {
			if err := actionDhcpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success create dhcp host!"})
		})
		host.Put("/update", func(c *routing.Context) error {
			if err := actionDhcpUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success update dhcp host!"})
		})
		host.Delete("/delete", func(c *routing.Context) error {
			if err := actionDhcpDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success delete dhcp host!"})
		})
	}

	if cfg.Smtp.Enabled {
		smtp := v0.Group("/smtp")

		smtp.Post("/config/download", func(c *routing.Context) error {
			if err := actionSmtpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success smtp download!"})
		})

		forward := smtp.Group("/forward")
		forward.Post("/create", func(c *routing.Context) error {
			if err := actionSmtpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success create smtp forward!"})
		})
		forward.Put("/update", func(c *routing.Context) error {
			if err := actionSmtpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success update smtp forward!"})
		})
		forward.Put("/rename", func(c *routing.Context) error {
			if err := actionSmtpForwardRename(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success rename smtp forward!"})
		})
		forward.Delete("/delete", func(c *routing.Context) error {
			if err := actionSmtpForwardDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success delete smtp forward!"})
		})

		user := smtp.Group("/user")
		user.Post("/create", func(c *routing.Context) error {
			if err := actionSmtpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success create smtp user!"})
		})
		user.Put("/update", func(c *routing.Context) error {
			if err := actionSmtpUserUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success update smtp user!"})
		})
		user.Delete("/delete", func(c *routing.Context) error {
			if err := actionSmtpUserDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success delete smtp user!"})
		})
	}

	if cfg.Squid.Enabled {
		squid := v0.Group("/squid")

		squid.Post("/config/download", func(c *routing.Context) error {
			if err := actionDownload(c, cfg.Squid.Path.Prod, cfg.Squid.Path.Temp); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := restartSquid(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success squid download!"})
		})
	}

	if cfg.Techmail.Enabled {
		tech := v0.Group("/techmail")

		tech.Post("/config/download", func(c *routing.Context) error {
			if err := actionDownload(c, cfg.Techmail.Path.Prod, cfg.Techmail.Path.Temp); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := newaliases(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success techmail download!"})
		})
	}

	if cfg.Samba.Enabled {
		samba := v0.Group("/samba")

		samba.Post("/config/download", func(c *routing.Context) error {
			if err := actionSambaDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := restartSamba(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success samba download!"})
		})

		share := samba.Group("/share")
		share.Post("/create", func(c *routing.Context) error {
			if err := actionSambaCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := restartSamba(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success create samba share!"})
		})
		share.Put("/quota", func(c *routing.Context) error {
			if err := actionSambaQuota(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success update samba share quota!"})
		})
		share.Delete("/delete", func(c *routing.Context) error {
			if err := actionSambaDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			if err := restartSamba(); err != nil {
				sentry.CaptureException(err)
				return c.Write(Response{500, err.Error()})
			}

			return c.Write(Response{200, "Success delete samba share!"})
		})
	}

	// serve index file
	router.Get("/", file.Content("ui/index.html"))
	// serve files under the "ui" subdirectory
	router.Get("/*", file.Server(file.PathMap{
		"/": "/ui/",
	}))

	http.Handle("/", router)
	_ = http.ListenAndServe(":"+cfg.Port, nil)
}

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

//func dhcpTestConfig(conf string) error {
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

func restartSquid() error {
	cli := exec.Command("/usr/sbin/squid", "-k", "reconfigure")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
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

func restartSamba() error {
	cli := exec.Command("/usr/bin/systemctl", "restart", "smb.service")
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
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

func actionDownload(c *routing.Context, prod string, temp string) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
			return err
		}

		head := temp + "/" + name + ".head"
		conf := prod + "/" + name
		if err := commit(head, recv, conf); err != nil {
			return err
		}
	}

	return nil
}

func dhcpCrud(line interface{}, download bool) error {
	temp := cfg.Dhcp.Path.Temp
	head := temp + "/dhcpd.conf.head"
	recv := temp + "/dhcpd.conf.recv"
	backup := temp + "/dhcpd.conf.recv.backup"
	conf := temp + "/dhcpd.conf"

	if err := beginTransaction(recv, backup); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	switch line.(type) {
	case string:
		if download {
			if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
				if err := rollback(recv, backup); err != nil {
					return err
				}
				return err
			}
		} else {
			if err := addDataToFile(recv, line.(string)); err != nil {
				if err := rollback(recv, backup); err != nil {
					return err
				}
				return err
			}
		}
	case map[string]string:
		if err := updateDataInFile(recv, line.(map[string]string)); err != nil {
			if err := rollback(recv, backup); err != nil {
				return err
			}
			return err
		}
	case []string:
		if err := deleteDataFromFile(recv, line.([]string)); err != nil {
			if err := rollback(recv, backup); err != nil {
				return err
			}
			return err
		}
	}

	if err := commit(head, recv, conf); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	//if err := dhcpTestConfig(tempConf); err != nil {
	//	if err := rollback(recv, backup); err != nil {
	//		return err
	//	}
	//	return err
	//}

	data, err := ioutil.ReadFile(conf)
	if err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	conf = cfg.Dhcp.Path.Prod + "/dhcpd.conf"
	if err := ioutil.WriteFile(conf, data, 0644); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	if err := restartDhcp(); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	return nil
}

func actionDhcpDownload(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	return dhcpCrud(data.Data["dhcpd"], true)
}

func actionDhcpCreate(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	return dhcpCrud(data.Data["dhcpd"], false)
}

func actionDhcpUpdate(c *routing.Context) error {
	var data dataMap
	if err := c.Read(&data); err != nil {
		return err
	}

	return dhcpCrud(data.Data["dhcpd"], false)
}

func actionDhcpDelete(c *routing.Context) error {
	var data dataSlice
	if err := c.Read(&data); err != nil {
		return err
	}

	return dhcpCrud(data.Data["dhcpd"], false)
}

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

func actionSmtpDownload(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := ioutil.WriteFile(recv, []byte(line.(string)), 0644); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	return nil
}

func actionSmtpCreate(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := addDataToFile(recv, line.(string)); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	return nil
}

func actionSmtpForwardRename(c *routing.Context) error {
	var data dataMap
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
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
			if err := updateDataInFile(recv, line); err != nil {
				return err
			}

			if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
				return err
			}
		}
	}

	return nil
}

func actionSmtpForwardDelete(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
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

	return nil
}

func actionSmtpUserUpdate(c *routing.Context) error {
	var data dataMap
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := updateDataInFile(recv, line); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	return nil
}

func actionSmtpUserDelete(c *routing.Context) error {
	var data dataSlice
	if err := c.Read(&data); err != nil {
		return err
	}

	lines := data.Data

	temp := cfg.Smtp.Path.Temp
	forward := cfg.Smtp.Path.Forward
	aliases := cfg.Smtp.Path.Aliases
	for name, line := range lines {
		recv := temp + "/" + name + ".recv"
		if err := deleteDataFromFile(recv, line); err != nil {
			return err
		}

		if err := smtpCommit(name, recv, temp, forward, aliases); err != nil {
			return err
		}
	}

	return nil
}

func actionSambaDownload(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	line := data.Data["samba"]

	temp := cfg.Samba.Path.Temp
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

	if err := ioutil.WriteFile(cfg.Samba.Path.Prod+"/smb.conf", confData, 0644); err != nil {
		return err
	}

	return nil
}

func actionSambaCreate(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	zfs := int(data.Data["is_zfs"].(float64))
	quota := data.Data["quota"]
	path := data.Data["path"]
	zfsPath := data.Data["zfs_path"]
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

	line := data.Data["samba"]

	temp := cfg.Samba.Path.Temp
	recv := temp + "/smb.conf.recv"
	backup := temp + "/smb.conf.recv.backup"
	if err := beginTransaction(recv, backup); err != nil {
		if err := rollback(recv, backup); err != nil {
			return err
		}
		return err
	}

	if err := addDataToFile(recv, line.(string)); err != nil {
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

	if err := ioutil.WriteFile(cfg.Samba.Path.Prod+"/smb.conf", confData, 0644); err != nil {
		return err
	}

	return nil
}

func actionSambaQuota(c *routing.Context) error {
	var data dataString
	if err := c.Read(&data); err != nil {
		return err
	}

	quota := data.Data["quota"]
	zfsPath := data.Data["zfs_path"]

	cli := exec.Command("/sbin/zfs", "set", "refquota="+quota.(string), zfsPath.(string))
	output, err := cli.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}

	return nil
}

func addDataToFile(recv string, data string) error {
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

func updateDataInFile(recv string, data map[string]string) error {
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

func deleteDataFromFile(recv string, data []string) error {
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
