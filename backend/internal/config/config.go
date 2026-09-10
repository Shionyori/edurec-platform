package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 是应用的顶层配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Engine    EngineConfig    `mapstructure:"engine"`
	Bilibili  BilibiliConfig  `mapstructure:"bilibili"`
}

// EngineConfig edurec-engine 接入配置
type EngineConfig struct {
	RecommendationsFile string `mapstructure:"recommendations_file"` // engine 输出的推荐结果 JSON 路径（路线 B 导入用）
	DatasetDir          string `mapstructure:"dataset_dir"`          // engine 演示/模拟数据集目录（demo_seed 播种用）
	SnapshotDir         string `mapstructure:"snapshot_dir"`         // 数据快照导出目录（export_snapshot 输出，engine 训练输入）
}

// BilibiliConfig 在线 B 站采集配置（搜索/评论实时爬取，见 backend/crawler/online.py）
type BilibiliConfig struct {
	PythonPath   string `mapstructure:"python_path"`   // Python 解释器，默认 "python"
	CrawlerDir   string `mapstructure:"crawler_dir"`   // backend/crawler 目录（online.py 所在，相对 server 运行目录 backend/）
	Category     string `mapstructure:"category"`      // 搜索落库分类名
	SearchLimit  int    `mapstructure:"search_limit"`  // 单次搜索导入条数
	CommentLimit int    `mapstructure:"comment_limit"` // 单次评论抓取条数
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig MySQL 数据库配置
type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// DSN 返回 MySQL 连接字符串
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database, d.Charset)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr 返回 Redis 地址
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	AccessSecret  string `mapstructure:"access_secret"`
	RefreshSecret string `mapstructure:"refresh_secret"`
	AccessExpire  int    `mapstructure:"access_expire"`
	RefreshExpire int    `mapstructure:"refresh_expire"`
}

// Load 从 YAML 文件和环境变量加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 允许环境变量覆盖（将 . 替换为 _，前缀为空）
	// 例如 SERVER_PORT 会覆盖 server.port
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}
