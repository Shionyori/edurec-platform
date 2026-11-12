package service_test

import (
	"path/filepath"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// 契约 fixture 的唯一来源是 edurec-engine/contracts/v1/fixtures/。
// 本地副本由 scripts/check-contract-sync.sh 维护；这些测试保证 platform
// 的导入实现与契约样本一致，而不是各写各的。

const contractTestdata = "testdata"

func TestImportAcceptsCanonicalContractFixture(t *testing.T) {
	recs := &fakeRecommendationRepository{}
	runs := &fakeRecommendationRunRepository{}
	svc := service.NewRecommendationImportService(
		recs, runs,
		newFakeUsers(1, 2, 3),
		newFakeResources(101, 102, 103),
		filepath.Join(contractTestdata, "recommendations.contract.json"),
	)

	result, err := svc.Import()
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedUsers != 3 || result.SkippedUsers != 0 {
		t.Fatalf("user stats = %+v, want 3 imported / 0 skipped", result)
	}
	if result.ImportedResources != 5 || result.SkippedResources != 0 {
		t.Fatalf("resource stats = %+v, want 5 imported / 0 skipped", result)
	}
	if len(recs.replaceCalls) != 3 {
		t.Fatalf("Replace() calls = %d, want 3", len(recs.replaceCalls))
	}
	if len(runs.created) != 1 {
		t.Fatalf("runs created = %d, want 1", len(runs.created))
	}
	run := runs.created[0]
	if run.RunID != "20260101_000100" || run.SnapshotRunID != "20260101_000000" {
		t.Fatalf("run metadata = %+v", run)
	}
	if run.ModelName != "semantic_deterministic_two_tower" || run.GeneratedAt != 1767225700 {
		t.Fatalf("run model/time = %+v", run)
	}
}

func TestImportRejectsMalformedContractFixtures(t *testing.T) {
	cases := []struct {
		file    string
		wantErr bool
	}{
		{"bom.json", true},
		{"string_id.json", true},
		{"float_id.json", true},
		{"negative_id.json", true},
		{"extra_top_level.json", true},
		{"not_object.json", true},
		{"trailing_garbage.json", true},
		// 契约要求 engine 自行去重 / 只输出数字键；平台侧的行为是
		// 「非数字键静默忽略」「重复 ID 原样保留」，属于已声明的容错语义。
		{"non_numeric_key.json", false},
		{"duplicate_ids.json", false},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			svc := newImportService(
				&fakeRecommendationRepository{},
				newFakeUsers(1),
				newFakeResources(101),
				filepath.Join(contractTestdata, "contract_invalid", tc.file),
			)
			_, err := svc.Import()
			if tc.wantErr && err == nil {
				t.Fatalf("%s 应被拒绝，但 Import() 成功了", tc.file)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("%s 应被容错接受，但 Import() 失败: %v", tc.file, err)
			}
		})
	}
}

func TestImportDoesNotDeduplicateByDesign(t *testing.T) {
	recs := &fakeRecommendationRepository{}
	svc := newImportService(
		recs,
		newFakeUsers(1),
		newFakeResources(101),
		writeRecommendations(t, `{"1": [101, 101]}`),
	)

	result, err := svc.Import()
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	// 契约把去重责任放在 engine 出口；platform 不重排、不去重。
	if result.ImportedResources != 2 {
		t.Fatalf("ImportedResources = %d, want 2（平台不去重）", result.ImportedResources)
	}
}
