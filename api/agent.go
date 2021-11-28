package api

import (
	"agent/api/services"
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
	"log"
	"net/http"
	"os"
)

type appSettings struct {
	Port   string
	Sentry struct {
		Dsn string
	}
	services.Dhcp
	services.Smtp
	services.Squid
	services.Techmail
	services.Samba
}

type response struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

var (
	Settings appSettings
)

func ReadConfig(cfg *appSettings) error {
	confFile, err := os.Open("/etc/agent/agent.yml")
	if err != nil {
		return err
	}
	defer func(confFile *os.File) {
		err = confFile.Close()
		if err != nil {
		}
	}(confFile)

	decoder := yaml.NewDecoder(confFile)
	err = decoder.Decode(cfg)
	if err != nil {
		return err
	}
	err = envconfig.Process("", cfg)
	if err != nil {
		return err
	}

	services.DhcpSettings = Settings.Dhcp
	services.ShareSettings = Settings.Samba
	services.SmtpSettings = Settings.Smtp
	services.TechmailSettings = Settings.Techmail
	services.SquidSettings = Settings.Squid

	return nil
}

func Run() {
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

	if Settings.Dhcp.Enabled {
		dhcp := v0.Group("/dhcp")

		dhcp.Post("/config/download", func(c *routing.Context) error {
			if err := actionDhcpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success dhcp download!"})
		})

		network := dhcp.Group("/network")
		network.Post("/create", func(c *routing.Context) error {
			if err := actionDhcpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success create dhcp network!"})
		})
		network.Put("/update", func(c *routing.Context) error {
			if err := actionDhcpUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success update dhcp network!"})
		})
		network.Delete("/delete", func(c *routing.Context) error {
			if err := actionDhcpDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success delete dhcp network!"})
		})

		host := dhcp.Group("/host")
		host.Post("/create", func(c *routing.Context) error {
			if err := actionDhcpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success create dhcp host!"})
		})
		host.Put("/update", func(c *routing.Context) error {
			if err := actionDhcpUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success update dhcp host!"})
		})
		host.Delete("/delete", func(c *routing.Context) error {
			if err := actionDhcpDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success delete dhcp host!"})
		})
	} else {
		message := errors.New("Service dhcp is disabled in /etc/agent/agent.yml")
		sentry.CaptureException(message)
	}

	if Settings.Smtp.Enabled {
		smtp := v0.Group("/smtp")

		smtp.Post("/config/download", func(c *routing.Context) error {
			if err := actionSmtpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success smtp download!"})
		})

		forward := smtp.Group("/forward")
		forward.Post("/create", func(c *routing.Context) error {
			if err := actionSmtpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success create smtp forward!"})
		})
		forward.Put("/update", func(c *routing.Context) error {
			if err := actionSmtpDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success update smtp forward!"})
		})
		forward.Put("/rename", func(c *routing.Context) error {
			if err := actionSmtpForwardRename(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success rename smtp forward!"})
		})
		forward.Delete("/delete", func(c *routing.Context) error {
			if err := actionSmtpForwardDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success delete smtp forward!"})
		})

		user := smtp.Group("/user")
		user.Post("/create", func(c *routing.Context) error {
			if err := actionSmtpCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success create smtp user!"})
		})
		user.Put("/update", func(c *routing.Context) error {
			if err := actionSmtpUserUpdate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success update smtp user!"})
		})
		user.Delete("/delete", func(c *routing.Context) error {
			if err := actionSmtpUserDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success delete smtp user!"})
		})
	} else {
		message := errors.New("Service smtp is disabled in /etc/agent/agent.yml")
		sentry.CaptureException(message)
	}

	if Settings.Squid.Enabled {
		squid := v0.Group("/squid")

		squid.Post("/config/download", func(c *routing.Context) error {
			if err := actionSquidDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success squid download!"})
		})
	} else {
		message := errors.New("Service squid is disabled in /etc/agent/agent.yml")
		sentry.CaptureException(message)
	}

	if Settings.Techmail.Enabled {
		tech := v0.Group("/techmail")

		tech.Post("/config/download", func(c *routing.Context) error {
			if err := actionTechMailDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success techmail download!"})
		})
	} else {
		message := errors.New("Service techmail is disabled in /etc/agent/agent.yml")
		sentry.CaptureException(message)
	}

	if Settings.Samba.Enabled {
		samba := v0.Group("/samba")

		samba.Post("/config/download", func(c *routing.Context) error {
			if err := actionSambaDownload(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success samba download!"})
		})

		share := samba.Group("/share")
		share.Post("/create", func(c *routing.Context) error {
			if err := actionSambaCreate(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success create samba share!"})
		})
		share.Put("/quota", func(c *routing.Context) error {
			if err := actionSambaQuota(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success update samba share quota!"})
		})
		share.Delete("/delete", func(c *routing.Context) error {
			if err := actionSambaDelete(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success delete samba share!"})
		})
		share.Post("/backup", func(c *routing.Context) error {
			if err := actionSambaBackup(c); err != nil {
				sentry.CaptureException(err)
				return c.Write(response{500, err.Error()})
			}

			return c.Write(response{200, "Success backup samba server!"})
		})
	} else {
		message := errors.New("Service samba is disabled in /etc/agent/agent.yml")
		sentry.CaptureException(message)
	}

	// serve index file
	router.Get("/", file.Content("ui/index.html"))
	// serve files under the "ui" subdirectory
	router.Get("/*", file.Server(file.PathMap{
		"/": "ui/",
	}))

	http.Handle("/", router)
	_ = http.ListenAndServe(":"+Settings.Port, nil)
}
