package config_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
)

// 默认教育分区白名单必须覆盖实测确认属于教育内容的 B 站分区。
//
// 背景：B 站搜索结果会按关键词混入非教育内容（影视/娱乐/游戏/音乐等），
// 因此落库前用 DefaultEducationalTypenames 做白名单拦截。但 B 站对内容的
// 分区归类很不规整——大量正经课程被归进「日常」「运动文化」「竞技体育」等
// 非教学分区，技能类教程又落在「软件应用」「职业职场」。这个用例把实测确认
// 属于教育内容的分区钉住，防止后续有人收紧白名单时把正常课程一起拦掉。
func TestDefaultEducationalTypenamesCoversVerifiedEducationalSections(t *testing.T) {
	// 来自 2026-09-19 两处实测：清理后 edurec 库的分区分布，
	// 以及用教育关键词调用 B 站搜索返回的分区统计
	verifiedEducational := []string{
		"校园学习",     // 课程主力分区
		"计算机技术",    // 编程/算法课程
		"科学科普",     // 数学/物理解说
		"野生技能协会",   // 技能教程（实测含机器学习、ROS、数据分析）
		"人文历史",     // 通识（实测含解析几何史、雅思流程、心理学）
		"社科·法律·心理", // 通识（实测含概率论、心理学）
		"日常",       // 误分类重灾区：数学分析、泛函分析、高等代数均在此
		"数码",       // 实测含高等数学习题讲解
		"运动文化",     // 实测含数学建模
		"竞技体育",     // 实测含数学建模国赛
		"软件应用",     // 实测含机器学习/数据分析软件教程
		"职业职场",     // 实测含人工智能、数据分析课程
		"科工机械",     // 实测含人工智能硬件相关内容
		"财经商业",     // 实测含人工智能商业分析内容
	}

	allowed := make(map[string]bool, len(config.DefaultEducationalTypenames))
	for _, name := range config.DefaultEducationalTypenames {
		allowed[name] = true
	}

	for _, name := range verifiedEducational {
		if !allowed[name] {
			t.Errorf("默认白名单缺少教育分区 %q：该分区下有真实课程，收紧后会误杀", name)
		}
	}
}

// 反向确认：典型非教育分区，以及内容不可控的分区，不应出现在默认白名单里
func TestDefaultEducationalTypenamesExcludesNonEducationalSections(t *testing.T) {
	nonEducational := []string{
		"影视剪辑", "影视杂谈", "娱乐粉丝创作", "娱乐杂谈",
		"明星综合", "网络游戏", "音乐综合", "原创音乐",
		"预告·资讯", "其他",
	}

	allowed := make(map[string]bool, len(config.DefaultEducationalTypenames))
	for _, name := range config.DefaultEducationalTypenames {
		allowed[name] = true
	}

	for _, name := range nonEducational {
		if allowed[name] {
			t.Errorf("默认白名单不应包含非教育/不可控分区 %q", name)
		}
	}
}

// 配置未显式给出白名单时应回退到默认值；显式给出时以配置为准
func TestAllowedTypenamesOrDefault(t *testing.T) {
	t.Run("未配置时回退默认", func(t *testing.T) {
		cfg := config.BilibiliConfig{}
		got := cfg.AllowedTypenamesOrDefault()
		if len(got) != len(config.DefaultEducationalTypenames) {
			t.Fatalf("len = %d, want %d", len(got), len(config.DefaultEducationalTypenames))
		}
	})

	t.Run("显式配置时以配置为准", func(t *testing.T) {
		cfg := config.BilibiliConfig{AllowedTypenames: []string{"校园学习"}}
		got := cfg.AllowedTypenamesOrDefault()
		if len(got) != 1 || got[0] != "校园学习" {
			t.Fatalf("got = %v, want [校园学习]", got)
		}
	})
}

// 长合集/课程判定：每次导入都会对每条 B 站内容跑一次，口径要稳。
// duration 取自 B 站采集的 "分钟:秒"（见 crawler/collect.py 的 _format_duration）。
func TestIsLongCourse(t *testing.T) {
	rules := config.ContentRulesConfig{}.CourseRulesOrDefault()

	if rules.CourseMinMinutes != config.DefaultCourseMinMinutes {
		t.Fatalf("CourseMinMinutes = %d, want %d", rules.CourseMinMinutes, config.DefaultCourseMinMinutes)
	}

	cases := []struct {
		name     string
		title    string
		duration string
		want     bool
	}{
		{"合集 + 长时长", "北大丘维声教授清华高等代数课程1080P高清修复版(全151集)", "3535:14", true},
		{"全集 + 长时长", "高等代数 丘维声老师 超高清修复版（全集）", "3535:14", true},
		{"课程 + 长时长", "【官方中英】2025年公认最好的【吴恩达机器学习课程】附课件", "1384:00", true},
		{"精讲 + 长时长", "【数学分析】课后习题精讲 华东师范大学 第五版 考研复习", "6389:00", true},
		{"集数标注 + 长时长", "数学分析（第三版）-复旦大学-陈纪修教授1080P高清(全198集)", "9003:20", true},
		{"讲数标注 + 长时长", "大连理工大学 力学中的泛函分析与变分原理 全43讲", "985:30", true},
		// 弱关键词档（教程/讲解）：需 ≥ 600 分钟（10 小时）
		{"教程 + 49 小时", "【Autolabor初级教程】ROS机器人入门", "2955:00", true},
		{"讲解 + 23 小时", "【2026版机器学习】周志华机器学习亲讲-西瓜书全网最详讲解", "1413:45", true},
		{"教程但只有 2 小时", "某入门教程", "120:00", false},
		{"讲解但只有 6 小时", "某某讲解", "360:00", false},
		{"教程恰好 10 小时", "某系统教程", "600:00", true},
		{"教程差 1 分钟到 10 小时", "某系统教程", "599:59", false},
		// 时长不达标：B 站大量短片标题写着「教程/全集」，必须靠时长拦住
		{"教程但只有 17 分钟", "2025新版【机器学习入门教程】17分钟让你看懂所有机器学习算法", "17:30", false},
		{"合集但时长不足阈值", "线性代数合集（上）", "119:59", false},
		{"课程但时长不足", "数学系最重要的基础课程之一——抽象代数教材推荐", "9:00", false},
		{"边界正好等于阈值", "线性代数合集（上）", "120:00", true},
		// 长但没有课程/合集标记：不属于本规则，仍按 video 落库
		{"长时长但无任何标记", "2026数学建模国赛B题可视化", "5000:00", false},
		// 「17分钟」不得被当成集数标注（曾因宽松正则误判）
		{"短片标题含「17分钟」", "17分钟让你看懂所有机器学习算法", "17:00", false},
		// 时长缺失或非法：不判为课程（宁可保守）
		{"课程但时长缺失", "某课程", "", false},
		{"课程但时长非法", "某课程", "abc", false},
		{"空标题", "", "5000:00", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rules.IsLongCourse(tc.title, tc.duration); got != tc.want {
				t.Errorf("IsLongCourse(%q, %q) = %v, want %v", tc.title, tc.duration, got, tc.want)
			}
		})
	}
}

// 规则可配置：改配置即改口径，不用改代码
func TestIsLongCourseRespectsCustomRules(t *testing.T) {
	rules := config.ContentRulesConfig{
		CourseMinMinutes:   60,
		CourseKeywords:     []string{"公开课"},
		CollectionMarkers:  []string{"专题"},
		LongformKeywords:   []string{"讲座"},
		LongformMinMinutes: 30,
	}

	if !rules.IsLongCourse("某公开课", "61:00") {
		t.Error("自定义关键词 + 达阈值应判为课程")
	}
	if rules.IsLongCourse("某公开课", "59:00") {
		t.Error("低于自定义阈值不应判为课程")
	}
	// 自定义规则替换了默认集合，默认关键词不再生效
	if rules.IsLongCourse("某课程", "500:00") {
		t.Error("自定义强词集合下默认关键词不应生效")
	}
	// 自定义弱词档门槛同样生效
	if !rules.IsLongCourse("某讲座", "31:00") {
		t.Error("自定义弱词 + 达其阈值应判为课程")
	}
	if rules.IsLongCourse("某讲座", "29:00") {
		t.Error("低于自定义弱词阈值不应判为课程")
	}
	// 默认弱词「教程」已被替换掉，不应再命中
	if rules.IsLongCourse("某教程", "900:00") {
		t.Error("自定义弱词集合下默认的「教程」不应生效")
	}
}

