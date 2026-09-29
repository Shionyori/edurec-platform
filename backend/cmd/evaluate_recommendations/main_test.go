package main

import "testing"

func TestRecallAtK(t *testing.T) {
	rel := map[uint]struct{}{1: {}, 3: {}}
	if got := recallAtK(rel, []uint{0, 1, 2, 3}, 3); got != 0.5 {
		t.Fatalf("recall@3 = %v, want 0.5", got)
	}
	if got := recallAtK(rel, []uint{1, 3, 0}, 5); got != 1.0 {
		t.Fatalf("recall@5 = %v, want 1.0", got)
	}
	if got := recallAtK(nil, []uint{1}, 5); got != 0 {
		t.Fatalf("empty relevant = %v, want 0", got)
	}
}

func TestHitRateAtK(t *testing.T) {
	rel := map[uint]struct{}{5: {}}
	if got := hitRateAtK(rel, []uint{0, 1, 2}, 3); got != 0 {
		t.Fatalf("hitrate@3 = %v, want 0", got)
	}
	if got := hitRateAtK(rel, []uint{0, 1, 5}, 3); got != 1 {
		t.Fatalf("hitrate@3 = %v, want 1", got)
	}
}

func TestNDCGAtK(t *testing.T) {
	rel := map[uint]struct{}{1: {}, 2: {}}
	// 命中在第 1、2 位应优于第 2、3 位
	best := ndcgAtK(rel, []uint{1, 2, 9}, 3)
	worse := ndcgAtK(rel, []uint{9, 1, 2}, 3)
	if !(best > worse) {
		t.Fatalf("ndcg best %v should exceed worse %v", best, worse)
	}
	if got := ndcgAtK(rel, []uint{9, 9, 9}, 3); got != 0 {
		t.Fatalf("ndcg with no hit = %v, want 0", got)
	}
}

func TestPopularityBaselineOrder(t *testing.T) {
	train := map[uint]map[uint]struct{}{
		1: {10: {}, 20: {}},
		2: {20: {}},
		3: {20: {}, 30: {}},
	}
	got := popularityBaseline(train)
	want := []uint{20, 10, 30} // 20 出现 3 次；10/30 同分按 id 升序
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("popularity[%d] = %d, want %d (full=%v)", i, got[i], want[i], got)
		}
	}
}
