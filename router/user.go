package router

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

// User 用户模型
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理登录请求
		type loginRequest struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		var req loginRequest

		// 绑定请求体
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"error": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 查询用户
		var user User
		err := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", req.Username).Scan(&user.ID, &user.Username, &user.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(401, gin.H{
					"error": "用户名或密码错误",
				})
				return
			}
			c.JSON(500, gin.H{
				"error": "数据库查询错误: " + err.Error(),
			})
			return
		}

		// TODO: 这里应该添加密码加密和验证的逻辑
		if user.Password != req.Password {
			c.JSON(401, gin.H{
				"error": "用户名或密码错误",
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "登录成功",
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
			},
		})
	}
}

func loadUserRoutes(r *gin.Engine, db *sql.DB) {
	// 用户相关路由组
	userGroup := r.Group("/users")
	{
		userGroup.POST("/login", loginHandler(db))
	}
}
