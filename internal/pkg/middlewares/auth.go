package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zihaolam/golang-media-upload-server/internal"
)

func AuthMiddleware(c *fiber.Ctx) error {
	// Get the Authorization header
	authToken := c.Get("Authorization")

	// Check if the Authorization header is empty
	if authToken == "" {
		// Return 401 Unauthorized
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Check if the value of the Authorization header is not "Bearer <your_token>"
	if authToken != internal.Env.SecretKey {
		// Return 401 Unauthorized
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// If everything is OK, call next
	return c.Next()
}
