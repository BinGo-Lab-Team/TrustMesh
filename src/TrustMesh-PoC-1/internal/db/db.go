package db

import (
	"TrustMesh-PoC-1/internal/constants"
	"TrustMesh-PoC-1/internal/table"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 包级变量
var (
	instance *gorm.DB
	once     sync.Once
)

// Path 数据库文件路径
func Path() string {
	path := filepath.Join(constants.ConfigDir, "data.db")
	return path
}

// GetDB 获取数据库对象，*必须* 先进行初始化
func GetDB() *gorm.DB {
	return instance
}

// InitDB 初始化数据库对象
func InitDB() (*gorm.DB, error) {
	var errMsg error = nil

	once.Do(func() {
		// 日志配置
		newLogger := logger.New(
			log.New(os.Stdout, "[GORM] ", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,  // Slow SQL threshold
				LogLevel:                  logger.Error, // Log level
				IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,         // Don't include params in the SQL log
				Colorful:                  false,        // Disable color
			},
		)

		// 连接数据库
		instance, errMsg = gorm.Open(sqlite.Open(Path()), &gorm.Config{
			Logger: newLogger,
		})

		if errMsg != nil {
			return
		}

		sqlDB, err := instance.DB()
		if err != nil {
			errMsg = err
			return
		}

		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)

		// 创建表结构
		errMsg = instance.AutoMigrate(&table.Peer{})
		if errMsg != nil {
			return
		}
	})

	return instance, errMsg
}
