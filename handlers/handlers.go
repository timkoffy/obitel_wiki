package handlers

import (
	"fmt"
	"net/http"
	"obitel-wiki/database"
	"obitel-wiki/middleware"
	"obitel-wiki/models"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// получение токена
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, _ := middleware.GenerateToken(user.ID, user.Role)
	// устанавливаем куки
	isProd := os.Getenv("APP_ENV") == "production"
	c.SetCookie("token", token, 3600*24*3, "/", "", isProd, true)
	
	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role})
}

// создание пользователя (редактора)
func CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required,oneof=editor admin"` // обычных юзеров создавать не надо, они и так читают
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{Username: req.Username, Password: string(hash), Role: req.Role}
	
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Username might be taken"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User created", "user": user})
}

// создание статьи (только editor/admin)
func CreateArticle(c *gin.Context) {
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	userID, _ := c.Get("user_id")
	article.AuthorID = userID.(uint)

	// GORM автоматически сохранит вложенные блоки и картинки благодаря каскаду
	if err := database.DB.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, article)
}

// список всех статей (доступно всем)
func GetArticles(c *gin.Context) {
	var articles []models.Article
	database.DB.Preload("Author").Find(&articles)
	c.JSON(http.StatusOK, articles)
}

// одна статья со всеми блоками и картинками (доступно всем)
func GetArticle(c *gin.Context) {
	var article models.Article
	if err := database.DB.Preload("Author").Preload("Blocks.Images").First(&article, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}
	c.JSON(http.StatusOK, article)
}

func UploadImage(c *gin.Context) {
	// получаем файл из формы
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не найден"})
		return
	}

	// проверяем расширение
	ext := filepath.Ext(file.Filename)
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый формат файла"})
		return
	}

	// создаем папку, если ее нет
	uploadDir := "./static/uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	// генерирм уникальное имя файла
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, filename)

	// сохраняем файл
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения файла"})
		return
	}

	// возвращаем url для вставки в текст
	c.JSON(http.StatusOK, gin.H{
		"url": "/static/uploads/" + filename,
	})
}

// обновление статьи (только editor/admin)
func UpdateArticle(c *gin.Context) {
	id := c.Param("id")
	
	var article models.Article
	if err := database.DB.Preload("Blocks").First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	// проверяем, что пользователь - админ
	role, _ := c.Get("role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit articles unless you're not an admin"})
		return
	}

	var input models.Article
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// обновляем заголовок
	article.Title = input.Title

	// удаляем старые блоки (каскадно удалятся и картинки)
	database.DB.Where("article_id = ?", article.ID).Delete(&models.Block{})

	// создаем новые блоки
	for i := range input.Blocks {
		input.Blocks[i].ID = 0
		input.Blocks[i].ArticleID = article.ID
		for j := range input.Blocks[i].Images {
			input.Blocks[i].Images[j].ID = 0
			input.Blocks[i].Images[j].BlockID = 0
		}
	}
	article.Blocks = input.Blocks

	if err := database.DB.Save(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, article)
}