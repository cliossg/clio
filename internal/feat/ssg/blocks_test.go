package ssg

import (
	"testing"

	"github.com/google/uuid"
)

func makeRenderedContent(id, sectionID uuid.UUID, kind, series string, seriesOrder int, tags []*Tag) *RenderedContent {
	return &RenderedContent{
		Content: &Content{
			ID:          id,
			SectionID:   sectionID,
			Kind:        kind,
			Series:      series,
			SeriesOrder: seriesOrder,
			Tags:        tags,
		},
	}
}

func TestGeneratedBlocksHasContent(t *testing.T) {
	tests := []struct {
		name   string
		blocks GeneratedBlocks
		want   bool
	}{
		{
			name:   "empty blocks",
			blocks: GeneratedBlocks{},
			want:   false,
		},
		{
			name: "has related content",
			blocks: GeneratedBlocks{
				Related: []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "", 0, nil)},
			},
			want: true,
		},
		{
			name: "has series next",
			blocks: GeneratedBlocks{
				SeriesNext: makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil),
			},
			want: true,
		},
		{
			name: "has series prev",
			blocks: GeneratedBlocks{
				SeriesPrev: makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil),
			},
			want: true,
		},
		{
			name: "has series index forward",
			blocks: GeneratedBlocks{
				SeriesIndexForward: []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil)},
			},
			want: true,
		},
		{
			name: "has series index backward",
			blocks: GeneratedBlocks{
				SeriesIndexBackward: []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil)},
			},
			want: true,
		},
		{
			name: "has all fields",
			blocks: GeneratedBlocks{
				Related:             []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "", 0, nil)},
				SeriesNext:          makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil),
				SeriesPrev:          makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil),
				SeriesIndexForward:  []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil)},
				SeriesIndexBackward: []RenderedContent{*makeRenderedContent(uuid.New(), uuid.New(), "blog", "s", 1, nil)},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.blocks.HasContent(); got != tt.want {
				t.Errorf("GeneratedBlocks.HasContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildBlocksDisabled(t *testing.T) {
	current := makeRenderedContent(uuid.New(), uuid.New(), "blog", "", 0, nil)
	allContent := []*RenderedContent{current}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: false})

	if blocks.HasContent() {
		t.Error("BuildBlocks with disabled config should return empty blocks")
	}
}

func TestBuildBlocksWithSeries(t *testing.T) {
	sectionID := uuid.New()
	seriesName := "test-series"

	post1 := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 1, nil)
	post2 := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 2, nil)
	post3 := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 3, nil)
	post4 := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 4, nil)

	allContent := []*RenderedContent{post1, post2, post3, post4}

	tests := []struct {
		name            string
		current         *RenderedContent
		wantPrev        bool
		wantNext        bool
		wantForwardLen  int
		wantBackwardLen int
	}{
		{
			name:            "first in series",
			current:         post1,
			wantPrev:        false,
			wantNext:        true,
			wantForwardLen:  3,
			wantBackwardLen: 0,
		},
		{
			name:            "middle in series",
			current:         post2,
			wantPrev:        true,
			wantNext:        true,
			wantForwardLen:  2,
			wantBackwardLen: 1,
		},
		{
			name:            "last in series",
			current:         post4,
			wantPrev:        true,
			wantNext:        false,
			wantForwardLen:  0,
			wantBackwardLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := BuildBlocks(tt.current, allContent, BlocksConfig{Enabled: true, MaxItems: 10})

			if (blocks.SeriesPrev != nil) != tt.wantPrev {
				t.Errorf("SeriesPrev present = %v, want %v", blocks.SeriesPrev != nil, tt.wantPrev)
			}
			if (blocks.SeriesNext != nil) != tt.wantNext {
				t.Errorf("SeriesNext present = %v, want %v", blocks.SeriesNext != nil, tt.wantNext)
			}
			if len(blocks.SeriesIndexForward) != tt.wantForwardLen {
				t.Errorf("SeriesIndexForward length = %d, want %d", len(blocks.SeriesIndexForward), tt.wantForwardLen)
			}
			if len(blocks.SeriesIndexBackward) != tt.wantBackwardLen {
				t.Errorf("SeriesIndexBackward length = %d, want %d", len(blocks.SeriesIndexBackward), tt.wantBackwardLen)
			}
		})
	}
}

func TestBuildBlocksSeriesWithMaxItems(t *testing.T) {
	sectionID := uuid.New()
	seriesName := "long-series"

	var posts []*RenderedContent
	for i := 1; i <= 10; i++ {
		posts = append(posts, makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, i, nil))
	}

	blocks := BuildBlocks(posts[4], posts, BlocksConfig{Enabled: true, MaxItems: 3})

	if len(blocks.SeriesIndexForward) > 3 {
		t.Errorf("SeriesIndexForward should be limited to 3, got %d", len(blocks.SeriesIndexForward))
	}
	if len(blocks.SeriesIndexBackward) > 3 {
		t.Errorf("SeriesIndexBackward should be limited to 3, got %d", len(blocks.SeriesIndexBackward))
	}
}

func TestBuildBlocksSeriesNotFound(t *testing.T) {
	sectionID := uuid.New()
	seriesName := "test-series"

	current := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 1, nil)
	otherPost := makeRenderedContent(uuid.New(), sectionID, "blog", seriesName, 2, nil)

	blocks := BuildBlocks(current, []*RenderedContent{otherPost}, BlocksConfig{Enabled: true, MaxItems: 5})

	if blocks.SeriesPrev != nil || blocks.SeriesNext != nil {
		t.Error("Should have no prev/next when current not in allContent")
	}
}

func TestBuildBlocksBlogContent(t *testing.T) {
	sectionID := uuid.New()
	tag1 := &Tag{ID: uuid.New(), Name: "golang"}
	tag2 := &Tag{ID: uuid.New(), Name: "testing"}

	current := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag1})
	relatedBlog := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag1})
	relatedArticle := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag1})
	unrelatedContent := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag2})

	allContent := []*RenderedContent{current, relatedBlog, relatedArticle, unrelatedContent}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 10})

	if len(blocks.Related) != 2 {
		t.Errorf("Expected 2 related items, got %d", len(blocks.Related))
	}

	for _, r := range blocks.Related {
		if r.ID == current.ID {
			t.Error("Related should not include current content")
		}
		if r.ID == unrelatedContent.ID {
			t.Error("Related should not include content without common tags")
		}
	}
}

func TestBuildBlocksBlogMultiSection(t *testing.T) {
	section1 := uuid.New()
	section2 := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), section1, "blog", "", 0, []*Tag{tag})
	sameSection := makeRenderedContent(uuid.New(), section1, "blog", "", 0, []*Tag{tag})
	otherSection := makeRenderedContent(uuid.New(), section2, "blog", "", 0, []*Tag{tag})

	allContent := []*RenderedContent{current, sameSection, otherSection}

	t.Run("multi section disabled", func(t *testing.T) {
		blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MultiSection: false, MaxItems: 10})

		for _, r := range blocks.Related {
			if r.ID == otherSection.ID {
				t.Error("Should not include content from other sections when MultiSection is false")
			}
		}
	})

	t.Run("multi section enabled", func(t *testing.T) {
		blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MultiSection: true, MaxItems: 10})

		found := false
		for _, r := range blocks.Related {
			if r.ID == otherSection.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should include content from other sections when MultiSection is true")
		}
	})
}

func TestBuildBlocksArticleContent(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag})
	relatedArticle := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag})
	relatedPost := makeRenderedContent(uuid.New(), sectionID, "post", "", 0, []*Tag{tag})
	relatedBlog := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag})

	allContent := []*RenderedContent{current, relatedArticle, relatedPost, relatedBlog}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 10})

	if len(blocks.Related) != 3 {
		t.Errorf("Expected 3 related items, got %d", len(blocks.Related))
	}
}

func TestBuildBlocksPostContent(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "post", "", 0, []*Tag{tag})
	relatedArticle := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag})

	allContent := []*RenderedContent{current, relatedArticle}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 10})

	if len(blocks.Related) != 1 {
		t.Errorf("Expected 1 related item, got %d", len(blocks.Related))
	}
}

func TestBuildBlocksMaxItems(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 3})

	if len(blocks.Related) > 3 {
		t.Errorf("Related should be limited to 3, got %d", len(blocks.Related))
	}
}

func TestBuildBlocksArticleMultiSection(t *testing.T) {
	section1 := uuid.New()
	section2 := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), section1, "article", "", 0, []*Tag{tag})
	sameSection := makeRenderedContent(uuid.New(), section1, "article", "", 0, []*Tag{tag})
	otherSection := makeRenderedContent(uuid.New(), section2, "article", "", 0, []*Tag{tag})

	allContent := []*RenderedContent{current, sameSection, otherSection}

	t.Run("multi section enabled", func(t *testing.T) {
		blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MultiSection: true, MaxItems: 10})

		found := false
		for _, r := range blocks.Related {
			if r.ID == otherSection.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should include content from other sections when MultiSection is true")
		}
	})
}

func TestBuildBlocksUnknownKind(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "unknown", "", 0, []*Tag{tag})
	related := makeRenderedContent(uuid.New(), sectionID, "unknown", "", 0, []*Tag{tag})

	allContent := []*RenderedContent{current, related}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 10})

	if len(blocks.Related) != 0 {
		t.Errorf("Unknown kind should not produce related content, got %d", len(blocks.Related))
	}
}

func TestHasCommonTags(t *testing.T) {
	tag1 := &Tag{ID: uuid.New(), Name: "golang"}
	tag2 := &Tag{ID: uuid.New(), Name: "testing"}
	tag3 := &Tag{ID: uuid.New(), Name: "web"}

	tests := []struct {
		name  string
		tags1 []*Tag
		tags2 []*Tag
		want  bool
	}{
		{
			name:  "both empty",
			tags1: nil,
			tags2: nil,
			want:  false,
		},
		{
			name:  "first empty",
			tags1: nil,
			tags2: []*Tag{tag1},
			want:  false,
		},
		{
			name:  "second empty",
			tags1: []*Tag{tag1},
			tags2: nil,
			want:  false,
		},
		{
			name:  "no common tags",
			tags1: []*Tag{tag1},
			tags2: []*Tag{tag2},
			want:  false,
		},
		{
			name:  "one common tag",
			tags1: []*Tag{tag1, tag2},
			tags2: []*Tag{tag2, tag3},
			want:  true,
		},
		{
			name:  "identical tags",
			tags1: []*Tag{tag1, tag2},
			tags2: []*Tag{tag1, tag2},
			want:  true,
		},
		{
			name:  "same tag by ID",
			tags1: []*Tag{tag1},
			tags2: []*Tag{{ID: tag1.ID, Name: "different name"}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1 := makeRenderedContent(uuid.New(), uuid.New(), "blog", "", 0, tt.tags1)
			c2 := makeRenderedContent(uuid.New(), uuid.New(), "blog", "", 0, tt.tags2)
			if got := hasCommonTags(c1, c2); got != tt.want {
				t.Errorf("hasCommonTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildBlocksArticleMaxItems(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	// Add many articles to trigger max items limit
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag}))
	}
	// Add blog posts too
	for i := 0; i < 5; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 3})

	if len(blocks.Related) > 3 {
		t.Errorf("Article related should be limited to 3, got %d", len(blocks.Related))
	}
}

func TestBuildBlocksArticleMaxItemsWithBlogFallback(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	// Add only 1 article, then 10 blog posts
	allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag}))
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 3})

	if len(blocks.Related) != 3 {
		t.Errorf("Expected 3 related items (1 article + 2 blogs), got %d", len(blocks.Related))
	}
}

func TestBuildBlocksArticleMultiSectionMaxItems(t *testing.T) {
	section1 := uuid.New()
	section2 := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), section1, "article", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	// Add content from other section
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), section2, "article", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MultiSection: true, MaxItems: 3})

	if len(blocks.Related) > 3 {
		t.Errorf("MultiSection article related should be limited to 3, got %d", len(blocks.Related))
	}
}

func TestBuildBlocksBlogMaxItemsWithArticleFallback(t *testing.T) {
	sectionID := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	// Add only 1 blog, then 10 articles
	allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "blog", "", 0, []*Tag{tag}))
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), sectionID, "article", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MaxItems: 3})

	if len(blocks.Related) != 3 {
		t.Errorf("Expected 3 related items (1 blog + 2 articles), got %d", len(blocks.Related))
	}
}

func TestBuildBlocksBlogMultiSectionMaxItems(t *testing.T) {
	section1 := uuid.New()
	section2 := uuid.New()
	tag := &Tag{ID: uuid.New(), Name: "golang"}

	current := makeRenderedContent(uuid.New(), section1, "blog", "", 0, []*Tag{tag})

	var allContent []*RenderedContent
	allContent = append(allContent, current)
	// Add content from other section
	for i := 0; i < 10; i++ {
		allContent = append(allContent, makeRenderedContent(uuid.New(), section2, "blog", "", 0, []*Tag{tag}))
	}

	blocks := BuildBlocks(current, allContent, BlocksConfig{Enabled: true, MultiSection: true, MaxItems: 3})

	if len(blocks.Related) > 3 {
		t.Errorf("MultiSection blog related should be limited to 3, got %d", len(blocks.Related))
	}
}

func TestLimitBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content []RenderedContent
		max     int
		wantLen int
	}{
		{
			name:    "empty slice",
			content: nil,
			max:     5,
			wantLen: 0,
		},
		{
			name:    "under limit",
			content: make([]RenderedContent, 3),
			max:     5,
			wantLen: 3,
		},
		{
			name:    "at limit",
			content: make([]RenderedContent, 5),
			max:     5,
			wantLen: 5,
		},
		{
			name:    "over limit",
			content: make([]RenderedContent, 10),
			max:     5,
			wantLen: 5,
		},
		{
			name:    "zero max",
			content: make([]RenderedContent, 5),
			max:     0,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitBlocks(tt.content, tt.max)
			if len(got) != tt.wantLen {
				t.Errorf("limitBlocks() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}
