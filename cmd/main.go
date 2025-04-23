package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YCLwow/gin-admin/internal/config"
	handlers "github.com/YCLwow/gin-admin/internal/handler"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// 数据库连接配置（建议放到环境变量）
const (
	dbDriver   = "mysql"
	dbUser     = "root"
	dbPassword = "123456"
	dbHost     = "localhost"
	dbPort     = "3306"
	dbName     = "ginadmin"
)

var db *sql.DB

func main() {
	// 1. 初始化数据库连接
	initDB()
	defer db.Close()

	// 2. 初始化Redis连接
	config.InitRedis()
	defer config.RedisClient.Close()

	// 3. 创建 Gin 实例
	router := setupRouter()

	// 4. 配置优雅关机
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	// 设置关机超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("强制关闭服务器:", err)
	}
	log.Println("服务器已正常关闭")
}

func initDB() {
	// 构建 DSN 连接字符串
	dsn := dbUser + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?parseTime=true"

	var err error
	db, err = sql.Open(dbDriver, dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	log.Println("成功连接到数据库")
}

func setupRouter() *gin.Engine {
	// Gin 模式配置
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 全局中间件
	router.Use(
		gin.Recovery(),  // 崩溃恢复
		requestLogger(), // 自定义请求日志
	)

	// 基础路由
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// 用户公开路由组
	userRoutes := router.Group("/users")
	{
		userRoutes.POST("/login", handlers.LoginHandler(db))
		userRoutes.POST("/", createUserHandler)
	}

	// 需要授权的路由组
	authRoutes := router.Group("/api")
	authRoutes.Use(handlers.AuthMiddleware())
	{
		authRoutes.POST("/logout", handlers.LogoutHandler())

		// 这里可以添加其他需要授权的路由
		authRoutes.GET("/profile", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			handlers.Success(c, gin.H{
				"user_id":  userID,
				"username": username,
			})
		})
	}

	return router
}

// 自定义请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		log.Printf("%s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}

func createUserHandler(c *gin.Context) {
	var newUser struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&newUser); err != nil {
		handlers.BadRequest(c, "无效的请求参数")
		return
	}

	result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
		newUser.Name, newUser.Email)

	if err != nil {
		handlers.InternalServerError(c, "创建用户失败")
		return
	}

	id, _ := result.LastInsertId()
	handlers.Success(c, gin.H{"id": id})
}
