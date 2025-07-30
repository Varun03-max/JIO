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

func JioTVServer(cfg JioTVServerConfig) error {
	// Load config (file or env)
	if err := config.Cfg.Load(cfg.ConfigPath); err != nil {
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

	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
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

	// Login routes (always enabled)
	app.Post("/login/sendOTP", handlers.LoginSendOTPHandler)
	app.Post("/login/verifyOTP", handlers.LoginVerifyOTPHandler)
	app.Post("/login", handlers.LoginPasswordHandler)

	// Load after login only
	if utils.FileExists("store.json") {
		if err := store.Init(); err != nil {
			return err
		}
		secureurl.Init()

		if config.Cfg.EPG || utils.FileExists("epg.xml.gz") {
			go epg.Init()
		}

		scheduler.Init()
		defer scheduler.Stop()

		// All protected routes
		app.Use("/out/", handlers.SLHandler)
		app.Get("/channels", handlers.ChannelsHandler)
		app.Get("/playlist.m3u", handlers.PlaylistHandler)
		app.Get("/live/:id", handlers.LiveHandler)
		app.Get("/live/:quality/:id", handlers.LiveQualityHandler)
		app.Get("/play/:id", handlers.PlayHandler)
		app.Get("/player/:id", handlers.PlayerHandler)
		app.Get("/render.m3u8", handlers.RenderHandler)
		app.Get("/render.ts", handlers.RenderTSHandler)
		app.Get("/render.key", handlers.RenderKeyHandler)
		app.Get("/mpd/:channelID", handlers.LiveMpdHandler)
		app.Post("/drm", handlers.DRMKeyHandler)
		app.Get("/render.mpd", handlers.MpdHandler)
		app.Use("/render.dash", handlers.DashHandler)
		app.Get("/epg.xml.gz", handlers.EPGHandler)
		app.Get("/epg/:channelID/:offset", handlers.WebEPGHandler)
		app.Get("/jtvimage/:file", handlers.ImageHandler)
		app.Get("/jtvposter/:date/:file", handlers.PosterHandler)
		app.Get("/dashtime", handlers.DASHTimeHandler)
		app.Get("/logout", handlers.LogoutHandler)
		handlers.Init()
	}

	// Always show index page
	app.Get("/", handlers.IndexHandler)
	app.Get("/favicon.ico", handlers.FaviconHandler)

	// Listen
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	if cfg.TLS {
		if cfg.TLSCertPath == "" || cfg.TLSKeyPath == "" {
			return fmt.Errorf("missing TLS cert/key")
		}
		return app.ListenTLS(addr, cfg.TLSCertPath, cfg.TLSKeyPath)
	}
	return app.Listen(addr)
}
