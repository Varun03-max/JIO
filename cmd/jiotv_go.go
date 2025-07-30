package cmd

import (
	"fmt"
	"net/http"

	"github.com/Varun03-max/JIO/internal/config"
	"github.com/Varun03-max/JIO/internal/constants"
	"github.com/Varun03-max/JIO/internal/handlers"
	"github.com/Varun03-max/JIO/internal/middleware"
	"github.com/Varun03-max/JIO/pkg/epg"
	"github.com/Varun03-max/JIO/pkg/scheduler"
	"github.com/Varun03-max/JIO/pkg/secureurl"
	"github.com/Varun03-max/JIO/pkg/store"
	"github.com/Varun03-max/JIO/pkg/utils"
	"github.com/Varun03-max/JIO/web"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
)

type JioTVServerConfig struct {
	Host        string
	Port        string
	ConfigPath  string
	TLS         bool
	TLSCertPath string
	TLSKeyPath  string
}

func JioTVServer(jiotvServerConfig JioTVServerConfig) error {
	if err := config.Cfg.Load(jiotvServerConfig.ConfigPath); err != nil {
		return err
	}

	utils.Log = utils.GetLogger()

	engine := html.NewFileSystem(http.FS(web.GetViewFiles()), ".html")
	if config.Cfg.Debug {
		engine.Reload(true)
	}

	app := fiber.New(fiber.Config{
		Views:             engine,
		Network:           fiber.NetworkTCP,
		StreamRequestBody: true,
		CaseSensitive:     false,
		StrictRouting:     false,
		EnablePrintRoutes: false,
		ServerHeader:      "JioTV Go",
		AppName:           fmt.Sprintf("JioTV Go %s", constants.Version),
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	app.Use(middleware.CORS())

	app.Use(logger.New(logger.Config{
		TimeZone: "Asia/Kolkata",
		Format:   "[${time}] ${status} - ${latency} ${method} ${path} Params:[${queryParams}] ${error}\n",
		Output:   utils.Log.Writer(),
	}))

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(web.GetStaticFiles()),
		PathPrefix: "static",
		Browse:     false,
	}))

	// Public login routes
	app.Post("/login/sendOTP", handlers.LoginSendOTPHandler)
	app.Post("/login/verifyOTP", handlers.LoginVerifyOTPHandler)
	app.Post("/login", handlers.LoginPasswordHandler)

	// Proceed only if session (store.json) is available
	loggedIn := utils.FileExists("store.json")
	if loggedIn {
		if err := store.Init(); err != nil {
			return err
		}
		secureurl.Init()
		if config.Cfg.EPG || utils.FileExists("epg.xml.gz") {
			go epg.Init()
		}
		scheduler.Init()
		defer scheduler.Stop()
		handlers.Init()
	}

	// All routes (many will fail if user is not logged in)
	app.Get("/", handlers.IndexHandler)
	app.Get("/logout", handlers.LogoutHandler)
	app.Get("/live/:id", handlers.LiveHandler)
	app.Get("/live/:quality/:id", handlers.LiveQualityHandler)
	app.Get("/render.m3u8", handlers.RenderHandler)
	app.Get("/render.ts", handlers.RenderTSHandler)
	app.Get("/render.key", handlers.RenderKeyHandler)
	app.Get("/channels", handlers.ChannelsHandler)
	app.Get("/playlist.m3u", handlers.PlaylistHandler)
	app.Get("/play/:id", handlers.PlayHandler)
	app.Get("/player/:id", handlers.PlayerHandler)
	app.Get("/favicon.ico", handlers.FaviconHandler)
	app.Get("/jtvimage/:file", handlers.ImageHandler)
	app.Get("/epg.xml.gz", handlers.EPGHandler)
	app.Get("/epg/:channelID/:offset", handlers.WebEPGHandler)
	app.Get("/jtvposter/:date/:file", handlers.PosterHandler)
	app.Get("/mpd/:channelID", handlers.LiveMpdHandler)
	app.Post("/drm", handlers.DRMKeyHandler)
	app.Get("/dashtime", handlers.DASHTimeHandler)
	app.Get("/render.mpd", handlers.MpdHandler)
	app.Use("/render.dash", handlers.DashHandler)
	app.Use("/out/", handlers.SLHandler)

	if jiotvServerConfig.TLS {
		if jiotvServerConfig.TLSCertPath == "" || jiotvServerConfig.TLSKeyPath == "" {
			return fmt.Errorf("TLS cert and key paths are required for HTTPS")
		}
		return app.ListenTLS(fmt.Sprintf("%s:%s", jiotvServerConfig.Host, jiotvServerConfig.Port), jiotvServerConfig.TLSCertPath, jiotvServerConfig.TLSKeyPath)
	}

	return app.Listen(fmt.Sprintf("%s:%s", jiotvServerConfig.Host, jiotvServerConfig.Port))
}
