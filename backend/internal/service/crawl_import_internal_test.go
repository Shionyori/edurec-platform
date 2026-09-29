package service

import "testing"

// B 站 metadata 的 duration 是 "16:08" / "1:30:00" 这类时钟串，需换算为分钟。
func TestParseClockDuration(t *testing.T) {
	cases := map[string]uint{
		"16:08":   16,
		"17:30":   18, // 四舍五入到分钟
		"1:30:00": 90,
		"0:30":    1,
		"":        0,
		"abc":     0,
		"12:xx":   0,
	}
	for in, want := range cases {
		if got := parseClockDuration(in); got != want {
			t.Errorf("parseClockDuration(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestMetadataDurationMinutesAcceptsSeconds(t *testing.T) {
	if got := metadataDurationMinutes(map[string]any{"duration": float64(5400)}); got != 90 {
		t.Errorf("numeric seconds = %d, want 90", got)
	}
	if got := metadataDurationMinutes(map[string]any{"duration": "1:30:00"}); got != 90 {
		t.Errorf("clock string = %d, want 90", got)
	}
	if got := metadataDurationMinutes(nil); got != 0 {
		t.Errorf("nil metadata = %d, want 0", got)
	}
}

func TestNormalizeDifficultyVariants(t *testing.T) {
	cases := map[string]string{
		"beginner":     "beginner",
		"Easy":         "beginner",
		"入门":           "beginner",
		"intermediate": "intermediate",
		"中等":           "intermediate",
		"advanced":     "advanced",
		"高级":           "advanced",
		"expert":       "", // 识别不了→未知
	}
	for raw, want := range cases {
		if got := normalizeDifficulty(raw); got != want {
			t.Errorf("normalizeDifficulty(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeDurationUnits(t *testing.T) {
	// 秒 → 分钟
	if got := normalizeDuration(float64(5400), "seconds"); got != 90 {
		t.Errorf("seconds = %d, want 90", got)
	}
	// 已是分钟则原样
	if got := normalizeDuration(float64(90), "minutes"); got != 90 {
		t.Errorf("minutes = %d, want 90", got)
	}
	if got := normalizeDuration("abc", "seconds"); got != 0 {
		t.Errorf("invalid = %d, want 0", got)
	}
}
