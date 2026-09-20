package config

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/spf13/viper"
)

// Config 是应用的顶层配置
type Config struct {
	Server        ServerConfig             `mapstructure:"server"`
	Database      DatabaseConfig           `mapstructure:"database"`
	Redis         RedisConfig              `mapstructure:"redis"`
	JWT           JWTConfig                `mapstructure:"jwt"`
	Engine        EngineConfig             `mapstructure:"engine"`
	Bilibili      BilibiliConfig           `mapstructure:"bilibili"`
	ContentRules  ContentRulesConfig       `mapstructure:"content_rules"`
	Datasets      map[string]DatasetConfig `mapstructure:"datasets"`
}

// 数据集字段映射支持的内部字段名。出现词表外的 key 一律报错：
// 拼错的 key 会静默产出空标题，从而让整批数据被跳过而看不出原因。
const (
	DatasetFieldTitle       = "title"
	DatasetFieldDescription = "description"
	DatasetFieldCoverURL    = "cover_url"
	DatasetFieldAuthor      = "author"
	DatasetFieldSourceURL   = "source_url"
	DatasetFieldCategory    = "category"
	DatasetFieldTags        = "tags"
	DatasetFieldViewCount   = "view_count"
)

// DatasetFormat 数据集文件的容器格式
const (
	DatasetFormatAuto  = "auto"
	DatasetFormatJSON  = "json"
	DatasetFormatJSONL = "jsonl"
	DatasetFormatCSV   = "csv"
)

// datasetFieldNames 映射词表，供校验与文档共用
var datasetFieldNames = []string{
	DatasetFieldTitle, DatasetFieldDescription, DatasetFieldCoverURL, DatasetFieldAuthor,
	DatasetFieldSourceURL, DatasetFieldCategory, DatasetFieldTags, DatasetFieldViewCount,
}

// DatasetConfig 第三方数据集导入配置：声明式地把外部字段映射到平台的采集结果契约。
// 数据集格式各家不同且会换，故字段对应关系全部外置到配置，换数据集只改 YAML 不改代码。
type DatasetConfig struct {
	File              string              `mapstructure:"file"`                // 数据集文件路径（相对 server 运行目录 backend/）
	Format            string              `mapstructure:"format"`              // auto | json | jsonl | csv，默认 auto 按扩展名与内容判定
	ItemsPath         string              `mapstructure:"items_path"`          // JSON 内记录数组所在路径（点号，如 data.list），空表示根即数组
	ResourceType      string              `mapstructure:"resource_type"`       // 落库的 resources.type：course | article | video
	DefaultCategory   string              `mapstructure:"default_category"`    // 分类字段为空时的兜底分类名
	SourceURLTemplate string              `mapstructure:"source_url_template"` // source_url 缺失时的兜底模板，{外部字段名} 作占位，如 https://x/learn/{obj_id}
	TagSeparator      string              `mapstructure:"tag_separator"`       // tags 为字符串时的分隔符，默认 ","
	Fields            map[string][]string `mapstructure:"fields"`              // 内部字段 → 外部字段候选，按序回退取第一个非空
	Metadata          []string            `mapstructure:"metadata"`            // 原样收进 metadata 的外部字段名
}

// Validate 校验单个数据集配置；配置错误必须在启动/导入前暴露，不能等到数据落库失败
func (d DatasetConfig) Validate(name string) error {
	if strings.TrimSpace(d.File) == "" {
		return fmt.Errorf("datasets.%s 缺少 file", name)
	}
	if !model.IsValidResourceType(d.ResourceType) {
		return fmt.Errorf("datasets.%s 的 resource_type %q 非法，只能是 course / article / video",
			name, d.ResourceType)
	}
	switch d.Format {
	case "", DatasetFormatAuto, DatasetFormatJSON, DatasetFormatJSONL, DatasetFormatCSV:
	default:
		return fmt.Errorf("datasets.%s 的 format %q 非法，只能是 auto / json / jsonl / csv", name, d.Format)
	}
	if len(d.Fields) == 0 {
		return fmt.Errorf("datasets.%s 缺少 fields 字段映射", name)
	}
	for field := range d.Fields {
		if !slices.Contains(datasetFieldNames, field) {
			return fmt.Errorf("datasets.%s 的 fields 含未知字段 %q，可用：%s",
				name, field, strings.Join(datasetFieldNames, " / "))
		}
	}
	return nil
}

// TagSeparatorOrDefault tags 为字符串时使用的分隔符
func (d DatasetConfig) TagSeparatorOrDefault() string {
	if d.TagSeparator == "" {
		return ","
	}
	return d.TagSeparator
}

// FormatOrDefault 归一化 format 取值
func (d DatasetConfig) FormatOrDefault() string {
	if d.Format == "" {
		return DatasetFormatAuto
	}
	return d.Format
}

// EngineConfig edurec-engine 接入配置
type EngineConfig struct {
	RecommendationsFile string `mapstructure:"recommendations_file"` // engine 输出的推荐结果 JSON 路径（路线 B 导入用）
	DatasetDir          string `mapstructure:"dataset_dir"`          // engine 演示/模拟数据集目录（demo_seed 播种用）
	SnapshotDir         string `mapstructure:"snapshot_dir"`         // 数据快照导出目录（export_snapshot 输出，engine 训练输入）
}

// BilibiliConfig 在线 B 站采集配置（搜索/评论实时爬取，见 backend/crawler/online.py）
type BilibiliConfig struct {
	PythonPath     string `mapstructure:"python_path"`      // Python 解释器，默认 "python"
	CrawlerDir     string `mapstructure:"crawler_dir"`      // backend/crawler 目录（online.py 所在，相对 server 运行目录 backend/）
	Category       string `mapstructure:"category"`         // 搜索落库分类名
	SearchLimit    int    `mapstructure:"search_limit"`     // 单次搜索导入条数
	SearchMaxPages int    `mapstructure:"search_max_pages"` // 无限滚动时最多翻 B 站页数，防无界爬取
	CommentLimit   int    `mapstructure:"comment_limit"`    // 单次评论抓取条数
	// AllowedTypenames 允许入库的 B 站分区名白名单。B 站搜索结果按关键词返回，
	// 非教育分区（影视剪辑/娱乐/游戏/音乐等）会随关键词一起被抓进来，必须在落库前拦掉。
	// 留空表示不限制（不推荐）。见 DefaultEducationalTypenames。
	AllowedTypenames []string `mapstructure:"allowed_typenames"`
}

// DefaultEducationalTypenames 教育向的 B 站分区白名单（配置未显式给出 allowed_typenames 时生效）。
//
// 取值依据（2026-09-19 实测两部分数据）：
//   - 清理后 edurec 库里保留资源的真实分区分布；
//   - 用「机器学习/雅思/高等数学/数据结构/数学/人工智能/数据分析/公开课/心理学/线性代数」
//     等教育关键词实际调用 B 站搜索，统计返回条目的分区。
//
// 收录原则：教学与知识类，以及 B 站对正经课程的高频误分类分区。
//   - 校园学习 / 计算机技术 / 科学科普 / 野生技能协会：明确的教学与知识分区；
//   - 人文历史 / 社科·法律·心理：通识学科内容；
//   - 日常 / 数码 / 运动文化 / 竞技体育：误分类重灾区（实测「数学分析」「泛函分析」
//     「高等代数」「数学建模」等课程被归入这些分区）；
//   - 软件应用 / 职业职场 / 科工机械 / 财经商业：技能与职业教育向
//     （实测「机器学习」「数据分析」「人工智能」的教程落在这里）。
//
// 刻意不收录：「其他」内容不可控；「预告·资讯」「原创音乐」等非教学分区。
// 未收录的分区一律不导入——新分区默认拦住，宁可漏也不要放回非教育内容。
var DefaultEducationalTypenames = []string{
	"校园学习",
	"计算机技术",
	"科学科普",
	"野生技能协会",
	"人文历史",
	"社科·法律·心理",
	"日常",
	"数码",
	"运动文化",
	"竞技体育",
	"软件应用",
	"职业职场",
	"科工机械",
	"财经商业",
}

// AllowedTypenamesOrDefault 返回生效的 B 站分区白名单：
// 配置里显式给了就用配置的，否则回退到 DefaultEducationalTypenames。
// 两者都为空表示不限制分区（不推荐）。
func (b BilibiliConfig) AllowedTypenamesOrDefault() []string {
	if len(b.AllowedTypenames) == 0 {
		return DefaultEducationalTypenames
	}
	return b.AllowedTypenames
}

// ContentRulesConfig 内容归类规则：判断一条 B 站内容是否算「长合集/系统课程」，
// 从而在导入时落库为 course（否则为 video）。全部可配置，改 YAML 即生效、不用改代码。
type ContentRulesConfig struct {
	// CourseMinMinutes 判定为课程的最短时长（分钟）。duration 形如 "2809:53"（分钟:秒）
	CourseMinMinutes int `mapstructure:"course_min_minutes"`
	// CourseKeywords 强关键词：命中即视为课程（子串匹配，如「课程」「精讲」）
	CourseKeywords []string `mapstructure:"course_keywords"`
	// CollectionMarkers 合集/系列标记（如「合集」「全集」「全N集」）；命中时只要时长达标也算课程
	CollectionMarkers []string `mapstructure:"collection_markers"`
	// LongformKeywords 弱关键词：「教程」「讲解」这类太泛、短片标题也常用的词。
	// 它们只在时长达到 LongformMinMinutes（默认 10 小时）时才判为课程，
	// 避免「17分钟让你看懂XX教程」这种短片被误判。
	LongformKeywords []string `mapstructure:"longform_keywords"`
	// LongformMinMinutes 弱关键词所需的最短时长（分钟）
	LongformMinMinutes int `mapstructure:"longform_min_minutes"`
}

// 默认规则。实测校准（2026-09-19）：
//   - 单看时长会把 174 条都算课程，连没写「课程」字样的长课也算；
//   - 单看关键词会把 17 分钟的短片教程误判为课程（14 条）；
//   - 「关键词且时长 >= 120 分钟」命中 52 条，逐条核对均为长课程/系列。
const (
	DefaultCourseMinMinutes = 120
	// DefaultLongformMinMinutes 弱关键词（教程/讲解）所需的时长：10 小时。
	// 取 600 是因为「教程」在 B 站短片标题里极常见，但能到 10 小时的必然是系统课程。
	DefaultLongformMinMinutes = 600
)

var (
	// DefaultCourseKeywords 只收「课程」「精讲」这类在 B 站细分领域里基本专指系统课的词。
	//
	// 刻意不收「教程」「讲解」进这一档：实测它们太泛，17 分钟短片也普遍这么写标题
	//（如「17分钟让你看懂所有机器学习算法」写成【入门教程】）。用「课程|精讲」时：
	// 时长 ≥ 120 分钟命中 50 条，而含这两个词却不足 120 分钟的仅 4 条，
	// 且全是「盘点课程」「课后习题精讲」这类非课程内容，正好被时长条件挡住。
	DefaultCourseKeywords = []string{"课程", "精讲"}

	DefaultCollectionMarkers = []string{"合集", "全集", "全套", "系列"}

	// DefaultLongformKeywords 弱关键词档：需要 ≥ 10 小时才判为课程。
	// 实测（2026-09-19）这批额外收进 13 条 10 小时以上的长课，
	// 包括 ROS 入门教程(49h)、高等代数讲解(32h)、周志华西瓜书讲解(23.5h)、
	// 雅思词汇带背(19h)、离散数学讲解(24.6h)。
	DefaultLongformKeywords = []string{"教程", "讲解"}
)

// CourseRulesOrDefault 返回生效的内容归类规则，未配置项回退到默认值
func (r ContentRulesConfig) CourseRulesOrDefault() ContentRulesConfig {
	out := r
	if out.CourseMinMinutes <= 0 {
		out.CourseMinMinutes = DefaultCourseMinMinutes
	}
	if len(out.CourseKeywords) == 0 {
		out.CourseKeywords = DefaultCourseKeywords
	}
	if len(out.CollectionMarkers) == 0 {
		out.CollectionMarkers = DefaultCollectionMarkers
	}
	if len(out.LongformKeywords) == 0 {
		out.LongformKeywords = DefaultLongformKeywords
	}
	if out.LongformMinMinutes <= 0 {
		out.LongformMinMinutes = DefaultLongformMinMinutes
	}
	return out
}

// IsLongCourse 判断一条内容是否算「长合集 / 系统课程」。
//
// 判定分两档，两档都要求时长达标：
//   - 强标记（course_keywords / collection_markers / 「全N集」「共N讲」）≥ course_min_minutes；
//   - 弱关键词（longform_keywords，如「教程」「讲解」）≥ longform_min_minutes（默认 10 小时）。
//
// 时长条件不可省：B 站大量「17分钟看完XX教程」的短片标题里也带「教程」，
// 光看关键词会把它们误判成课程 —— 弱词档用更高的时长门槛解决这个问题。
//
// durationText 为 B 站采集的 duration 原值（"2809:53" 即 分钟:秒）。
func (r ContentRulesConfig) IsLongCourse(title, durationText string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	minutes := parseDurationMinutes(durationText)

	if minutes >= r.CourseMinMinutes {
		if hasCourseMarker(title, r.CourseKeywords) ||
			hasCourseMarker(title, r.CollectionMarkers) ||
			episodeMarkPattern.MatchString(title) {
			return true
		}
	}
	if minutes >= r.LongformMinMinutes && hasCourseMarker(title, r.LongformKeywords) {
		return true
	}
	return false
}

// episodeMarkPattern 匹配明确的集数/讲数标注：「全151集」「全198集」「共78集」「全39讲」。
//
// 两个刻意的写法约束：
//   - 集/讲前必须是**ASCII 数字**。用 \d 会连中文数字一起匹配 ——
//     「代**数**」「基础**课**程」里的「数」「课」都不是 [集讲]，但「…数学…集」这类组合会误命中；
//   - 只认带「全/共」前缀的写法，避免把「17分钟让你看懂…」里的「17分」当集数。
var episodeMarkPattern = regexp.MustCompile(`(?:全|共)\s*[0-9]+\s*[集讲]`)

func hasCourseMarker(title string, markers []string) bool {
	for _, marker := range markers {
		if marker != "" && strings.Contains(title, marker) {
			return true
		}
	}
	return false
}

// parseDurationMinutes 解析 B 站采集的时长字符串为分钟数。
//
// **两段是「分钟:秒」，不是「时:分」**：crawler/collect.py 的 _format_duration 把
// 接口返回的秒数格式化成 `f"{seconds // 60}:{seconds % 60:02d}"`，所以 9003:20 表示
// 9003 分钟。曾按「时:分」累乘（两段也 *60），把 119:59 算成 7199 分钟，
// 导致时长门槛形同失效。三段的 "时:分:秒" 也兼容。
// 无法解析或为空返回 0（0 不会通过阈值判定）。
func parseDurationMinutes(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	parts := strings.Split(text, ":")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < 0 {
			return 0
		}
		values = append(values, value)
	}
	switch len(values) {
	case 1: // "120" 视作分钟
		return values[0]
	case 2: // "分钟:秒"
		return values[0]
	case 3: // "时:分:秒"
		return values[0]*60 + values[1]
	default:
		return 0
	}
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

	if err := cfg.validateDatasets(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateDatasets 校验全部数据集配置。放在加载阶段是为了让拼错的字段映射
// 在启动时就报错，而不是等导入跑完才发现整批数据都因缺字段被跳过。
func (c *Config) validateDatasets() error {
	for name, dataset := range c.Datasets {
		if err := dataset.Validate(name); err != nil {
			return err
		}
	}
	return nil
}
