package router

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// 全局中间件
	// r.Use(gin.Logger())

	// 注册各业务模块路由
	loadUserRoutes(r, db) // 用户模块

	// 默认路由
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server is running"})
	})

	return r
}
