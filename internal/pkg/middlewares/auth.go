package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
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
	if authToken != strings.Replace(authToken, "Bearer ", "", 1) {
		// Return 401 Unauthorized
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// If everything is OK, call next
	return c.Next()
}
