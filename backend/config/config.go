package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config 全局配置结构体
type Config struct {
	Port        string
	Host        string
	MySQL       MySQLConfig
	Redis       RedisConfig
	InfluxDB    InfluxDBConfig
	EMQX        EMQXConfig
}

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// InfluxDBConfig InfluxDB 配置
type InfluxDBConfig struct {
	URL      string
	Token    string
	Org      string
	Bucket   string
}

// EMQXConfig EMQX Webhook 配置
type EMQXConfig struct {
	WebhookPath string
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// InitConfig 初始化配置，从 .env 文件加载
func InitConfig() error {
	// 尝试加载 .env 文件（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Printf("[WARN] 未找到 .env 文件，将使用系统环境变量: %v", err)
	}

	// 创建配置实例
	GlobalConfig = &Config{
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", "YourPassword123"),
			Database: getEnv("MYSQL_DATABASE", "hkbn_iot"),
		},

		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},

		InfluxDB: InfluxDBConfig{
			URL:      getEnv("INFLUXDB_URL", "http://localhost:8086"),
			Token:    getEnv("INFLUXDB_TOKEN", "my-super-token"),
			Org:      getEnv("INFLUXDB_ORG", "hkbn"),
			Bucket:   getEnv("INFLUXDB_BUCKET", "iot_health"),
		},

		EMQX: EMQXConfig{
			WebhookPath: getEnv("EMQX_WEBHOOK_PATH", "/api/v1/device/raw-tcp"),
		},
	}

	log.Printf("[INFO] 配置加载完成，服务将在 %s:%s 启动", GlobalConfig.Host, GlobalConfig.Port)
	return nil
}

// GetMySQLDSN 获取 MySQL DSN 连接字符串
func (c *Config) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQL.User,
		c.MySQL.Password,
		c.MySQL.Host,
		c.MySQL.Port,
		c.MySQL.Database,
	)
}

// GetRedisAddr 获取 Redis 地址
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// GetServerAddr 获取服务地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// getEnv 获取环境变量，支持默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetDeviceStateTTL 获取设备状态 TTL（5分钟）
func GetDeviceStateTTL() time.Duration {
	return 5 * time.Minute
}

// GetAlertTTL 获取告警状态 TTL（30分钟）
func GetAlertTTL() time.Duration {
	return 30 * time.Minute
}
