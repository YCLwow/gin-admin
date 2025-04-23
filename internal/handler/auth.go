// handlers/auth.go
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/YCLwow/gin-admin/internal/config"
	"github.com/YCLwow/gin-admin/internal/models"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		println("进来了")
		// 1. 绑定请求参数
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
			return
		}

		// 2. 查询用户是否存在
		var user models.User
		query := "SELECT id, username, password FROM users WHERE username = ?"
		row := db.QueryRow(query, req.Username)

		err := row.Scan(&user.ID, &user.Username, &user.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			} else {
				fmt.Printf("数据库查询错误: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误"})
			}
			return
		}

		// 3. 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			// 如果数据库密码是明文，可以先尝试哈希比较
			if user.Password != req.Password {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
				return
			}
			// 或者直接返回错误
			// c.JSON(http.StatusUnauthorized, gin.H{"error": "密码未加密存储，请联系管理员"})
			// return
		}
		fmt.Printf("成功查询到用户: ID=%d, Username=%s\n", user.ID, user.Username)
		fmt.Printf("用户信息: ID=%d, Username=%s, Password=%s\n", user.ID, user.Username, user.Password)
		println(req.Username, req.Password, row, err)

		// 4. 生成JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  user.ID,
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, err := token.SignedString([]byte(config.JWTSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成访问令牌"})
			return
		}

		// 5. 将token保存到Redis
		ctx := context.Background()
		if err := config.SaveToken(ctx, user.ID, user.Username, tokenString); err != nil {
			fmt.Printf("保存token到Redis失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存访问令牌"})
			return
		}

		// 6. 返回响应
		Success(c, gin.H{
			"token":   tokenString,
			"expires": time.Now().Add(time.Hour * 24).Format(time.RFC3339),
		})
	}
}

// 验证token的中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供授权令牌"})
			return
		}

		// 如果token带有Bearer前缀，去掉前缀
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// 解析token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			return
		}

		// 从token中获取claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 从Redis验证token
			username, _ := claims["username"].(string)
			ctx := context.Background()

			cachedToken, err := config.GetToken(ctx, username)
			if err != nil || cachedToken != tokenString {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "令牌已失效，请重新登录"})
				return
			}

			// 将用户信息存储到上下文中
			userID, _ := claims["user_id"].(float64)
			c.Set("user_id", int(userID))
			c.Set("username", username)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			return
		}
	}
}

// 注销登录
func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未找到用户信息"})
			return
		}

		// 从Redis中删除token
		ctx := context.Background()
		if err := config.InvalidateToken(ctx, username.(string)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "注销失败"})
			return
		}

		Success(c, gin.H{"message": "注销成功"})
	}
}
