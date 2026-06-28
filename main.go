package main

import (
	"log"
	"project173/config"
	"project173/database"
	"project173/handlers"
	"project173/middleware"
	"project173/models"
	"project173/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized successfully")

	jwtService := jwt.NewService(cfg.JWTSecret)

	userHandler := handlers.NewUserHandler(jwtService)
	propertyHandler := handlers.NewPropertyHandler()
	orderHandler := handlers.NewOrderHandler()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		api.GET("/properties", propertyHandler.List)
		api.GET("/properties/:id", propertyHandler.Get)

		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddleware(jwtService))
		{
			authenticated.GET("/auth/me", userHandler.Me)

			landlordRoutes := authenticated.Group("")
			landlordRoutes.Use(middleware.RoleMiddleware(string(models.RoleLandlord)))
			{
				landlordRoutes.POST("/properties", propertyHandler.Create)
				landlordRoutes.PUT("/properties/:id", propertyHandler.Update)
				landlordRoutes.DELETE("/properties/:id", propertyHandler.Delete)
				landlordRoutes.PATCH("/properties/:id/status", propertyHandler.SetStatus)
			}

			authenticated.POST("/orders", orderHandler.Create)
			authenticated.GET("/orders", orderHandler.List)
			authenticated.GET("/orders/:id", orderHandler.Get)
			authenticated.POST("/orders/:id/transition", orderHandler.Transition)
		}
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
