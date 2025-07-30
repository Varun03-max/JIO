func JioTVServer(jiotvServerConfig JioTVServerConfig) error {
	// Load the config file or ENV
	if err := config.Cfg.Load(jiotvServerConfig.ConfigPath); err != nil {
		return err
	}

	// Initialize logger
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

	// Routes for login — always available
	app.Post("/login/sendOTP", handlers.LoginSendOTPHandler)
	app.Post("/login/verifyOTP", handlers.LoginVerifyOTPHandler)
	app.Post("/login", handlers.LoginPasswordHandler)

	// Check if store.json (user session) exists before initializing login-dependent modules
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
	}

	// Initialize the television object (safe — will skip if not logged in)
	handlers.Init()

	// Static routes
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

	// Launch server
	if jiotvServerConfig.TLS {
		if jiotvServerConfig.TLSCertPath == "" || jiotvServerConfig.TLSKeyPath == "" {
			return fmt.Errorf("TLS cert and key paths are required for HTTPS. Please provide them using --tls-cert and --tls-key flags")
		}
		return app.ListenTLS(fmt.Sprintf("%s:%s", jiotvServerConfig.Host, jiotvServerConfig.Port), jiotvServerConfig.TLSCertPath, jiotvServerConfig.TLSKeyPath)
	}

	return app.Listen(fmt.Sprintf("%s:%s", jiotvServerConfig.Host, jiotvServerConfig.Port))
}
