package model_test

import (
	"sync"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm/schema"
)

// FindBySourceURLs 在在线翻页的每次请求上执行 WHERE source_url IN (...)，
// 没有索引就会退化为全表扫描，而这是无限滚动路径上的热查询。
func TestResourceSourceURLIsIndexed(t *testing.T) {
	parsed, err := schema.Parse(&model.Resource{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("解析 Resource schema 失败: %v", err)
	}

	for _, idx := range parsed.ParseIndexes() {
		for _, opt := range idx.Fields {
			if opt.Field != nil && opt.Field.DBName == "source_url" {
				return
			}
		}
	}
	t.Fatal("source_url 缺少索引，FindBySourceURLs 会退化为全表扫描")
}
