package main

import (
	"obitel-wiki/database"
	"obitel-wiki/handlers"
	"obitel-wiki/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	database.InitDB()
	r := gin.Default()

	frontendPort := database.GetEnv("FRONTEND_PORT", "8080")

	// статика для картинок и стилей
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// апи
	api := r.Group("/api")
	{
		// без авторизации
		api.POST("/login", handlers.Login)
		api.GET("/articles", handlers.GetArticles)
		api.GET("/articles/:id", handlers.GetArticle)

		// требуют авторизации
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/articles", middleware.RoleMiddleware("admin", "editor"), handlers.CreateArticle)
			protected.POST("/upload", middleware.RoleMiddleware("admin", "editor"), handlers.UploadImage)
			protected.PUT("/articles/:id", middleware.RoleMiddleware("admin", "editor"), handlers.UpdateArticle)

			// только для админа
			admin := protected.Group("/admin")
			admin.Use(middleware.RoleMiddleware("admin"))
			{
				admin.POST("/users", handlers.CreateUser)
			}
		}
	}

	// фронтенд (статический)
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.Run(":" + frontendPort)
}