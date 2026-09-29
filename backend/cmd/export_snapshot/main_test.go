package main

import (
	"testing"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// TestResourceHeaderIncludesDescription 钉住「快照必须导出资源正文」这一契约：
// engine 侧通过 DictReader 按列名读取 description，缺失时会静默降级为空串，
// 导致语义召回模型拿不到正文。列顺序由本测试锁定，避免以后改动时漏掉。
func TestResourceHeaderIncludesDescription(t *testing.T) {
	header := resourceHeader()
	want := []string{"resource_id", "title", "description", "type", "category_id",
		"tags_json", "metadata_json", "avg_rating", "view_count", "created_at"}
	if len(header) != len(want) {
		t.Fatalf("表头列数 %d，期望 %d：%v", len(header), len(want), header)
	}
	for i := range want {
		if header[i] != want[i] {
			t.Fatalf("第 %d 列 = %q，期望 %q", i, header[i], want[i])
		}
	}
}

func TestResourceRowsMapsDescription(t *testing.T) {
	created := time.Unix(1700000000, 0)
	res := []model.Resource{{
		Model:       gorm.Model{ID: 7, CreatedAt: created},
		Title:       "线性代数",
		Description: "矩阵与向量空间入门",
		Type:        model.ResourceTypeVideo,
		CategoryID:  3,
		Tags:        `["数学"]`,
		Metadata:    `{"duration":600}`,
		AvgRating:   4.5,
		ViewCount:   120,
	}}

	header := resourceHeader()
	rows := resourceRows(res)
	if len(rows) != 1 {
		t.Fatalf("期望 1 行，实际 %d", len(rows))
	}
	if len(rows[0]) != len(header) {
		t.Fatalf("行长 %d 与表头列数 %d 不一致", len(rows[0]), len(header))
	}

	idx := -1
	for i, h := range header {
		if h == "description" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("表头缺少 description 列")
	}
	if rows[0][idx] != "矩阵与向量空间入门" {
		t.Fatalf("description 列 = %q，期望 %q", rows[0][idx], "矩阵与向量空间入门")
	}
}

// TestResourceRowsEmptyTags 保持既有语义：空 tags 导出为 []，而非空串。
func TestResourceRowsEmptyTags(t *testing.T) {
	header := resourceHeader()
	rows := resourceRows([]model.Resource{{Model: gorm.Model{ID: 1}, Title: "t", Tags: ""}})
	idx := -1
	for i, h := range header {
		if h == "tags_json" {
			idx = i
			break
		}
	}
	if rows[0][idx] != "[]" {
		t.Fatalf("空 tags 应导出为 []，实际 %q", rows[0][idx])
	}
}
