package api

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/zihaolam/golang-media-upload-server/internal"
)

type api struct {
	app *fiber.App
}

type Handler = func(c *fiber.Ctx) error

func NewApi() *api {
	return &api{
		app: fiber.New(fiber.Config{
			BodyLimit: 40 * 1024 * 1024, // this is the default limit of 4MB
		}),
	}
}

func setupCORS(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin,Content-Type,Accept,Content-Length,Accept-Language,Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOrigins:     "http://localhost:3059",
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,OPTIONS",
	}))
}

func setupLogger(app *fiber.App) {
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] (${ip}:${port}) ${status} - ${method} ${path}\n",
		TimeFormat: "2006-Jan-02 03:04:05PM",
	}))
}

func (a *api) Setup() {
	setupCORS(a.app)
	setupLogger(a.app)
	a.RegisterRoutes()
	a.Serve()
}

func (a *api) Serve() {
	err := a.app.Listen(fmt.Sprintf("%s:%s", internal.Env.Host, internal.Env.Port))
	if err != nil {
		log.Fatal(err)
	}
}

func (a *api) RegisterRoutes() {
	a.RegisterUtilRoutes()

	transcodeApi := NewTranscodeApi(a)
	transcodeApi.RegisterRoutes(a)
}

func (a *api) GetV1Group() fiber.Router {
	return a.app.Group("/v1")
}

func (a *api) RegisterUtilRoutes() {
	a.app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("Pong!")
	})
}
