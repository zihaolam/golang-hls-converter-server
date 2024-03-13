package main

import (
	"log"

	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	runtime.GOMAXPROCS(10)
	app := fiber.New(fiber.Config{
		BodyLimit: 300 * 1024 * 1024, // this is the default limit of 4MB
	})
	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin,Content-Type,Accept,Content-Length,Accept-Language,Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOrigins:     "http://localhost:3059",
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,OPTIONS",
	}))

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("Pong!")
	})

	app.Post("/transcode", transcodeFileHandler)

	log.Fatal(app.Listen("localhost:4884"))
}
