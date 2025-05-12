package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/YCLwow/gin-admin/internal/config"
	"github.com/YCLwow/gin-admin/router"

	_ "github.com/go-sql-driver/mysql"
)

// 匿名导入驱动包

// 连接数据库
var db *sql.DB

func main() {
	// 初始化数据库
	initDB()
	defer db.Close()

	// 挂载路由
	r := router.SetupRouter(db)
	r.Run(":8080")
}

func initDB() {
	// 构建 DSN 连接字符串
	dsn := config.DBUser + ":" + config.DBPassword + "@tcp(" + config.DBHost + ":" + config.DBPort + ")/" + config.DBName + "?parseTime=true"

	var err error
	db, err = sql.Open(config.DBDriver, dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	log.Println("成功连接到数据库")
}
