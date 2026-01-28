package ssg

import (
	"sort"

	"github.com/google/uuid"
)

type GeneratedBlocks struct {
	Related []RenderedContent

	SeriesNext          *RenderedContent
	SeriesPrev          *RenderedContent
	SeriesIndexForward  []RenderedContent
	SeriesIndexBackward []RenderedContent
}

func (b *GeneratedBlocks) HasContent() bool {
	return len(b.Related) > 0 ||
		b.SeriesNext != nil ||
		b.SeriesPrev != nil ||
		len(b.SeriesIndexForward) > 0 ||
		len(b.SeriesIndexBackward) > 0
}

type BlocksConfig struct {
	Enabled      bool
	MultiSection bool
	MaxItems     int
}

func BuildBlocks(current *RenderedContent, allContent []*RenderedContent, cfg BlocksConfig) *GeneratedBlocks {
	blocks := &GeneratedBlocks{}

	if !cfg.Enabled {
		return blocks
	}

	if current.Series != "" {
		buildSeriesBlocks(blocks, current, allContent, cfg.MaxItems)
		return blocks
	}

	if current.Kind == "blog" {
		buildBlogBlocks(blocks, current, allContent, cfg)
	} else if current.Kind == "article" || current.Kind == "post" {
		buildArticleBlocks(blocks, current, allContent, cfg)
	}

	return blocks
}

func buildBlogBlocks(blocks *GeneratedBlocks, current *RenderedContent, allContent []*RenderedContent, cfg BlocksConfig) {
	added := make(map[uuid.UUID]bool)
	added[current.ID] = true

	// Priority 1: Blog posts in same section with matching tags
	for _, c := range allContent {
		if len(blocks.Related) >= cfg.MaxItems {
			break
		}
		if c.Kind == "blog" && c.SectionID == current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
			blocks.Related = append(blocks.Related, *c)
			added[c.ID] = true
		}
	}

	// Priority 2: Articles in same section with matching tags
	for _, c := range allContent {
		if len(blocks.Related) >= cfg.MaxItems {
			break
		}
		if (c.Kind == "article" || c.Kind == "post") && c.SectionID == current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
			blocks.Related = append(blocks.Related, *c)
			added[c.ID] = true
		}
	}

	// Priority 3: Content from other sections with matching tags (if enabled)
	if cfg.MultiSection {
		for _, c := range allContent {
			if len(blocks.Related) >= cfg.MaxItems {
				break
			}
			if c.SectionID != current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
				blocks.Related = append(blocks.Related, *c)
				added[c.ID] = true
			}
		}
	}
}

func buildArticleBlocks(blocks *GeneratedBlocks, current *RenderedContent, allContent []*RenderedContent, cfg BlocksConfig) {
	added := make(map[uuid.UUID]bool)
	added[current.ID] = true

	// Priority 1: Articles in same section with matching tags
	for _, c := range allContent {
		if len(blocks.Related) >= cfg.MaxItems {
			break
		}
		if (c.Kind == "article" || c.Kind == "post") && c.SectionID == current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
			blocks.Related = append(blocks.Related, *c)
			added[c.ID] = true
		}
	}

	// Priority 2: Blog posts in same section with matching tags
	for _, c := range allContent {
		if len(blocks.Related) >= cfg.MaxItems {
			break
		}
		if c.Kind == "blog" && c.SectionID == current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
			blocks.Related = append(blocks.Related, *c)
			added[c.ID] = true
		}
	}

	// Priority 3: Content from other sections with matching tags (if enabled)
	if cfg.MultiSection {
		for _, c := range allContent {
			if len(blocks.Related) >= cfg.MaxItems {
				break
			}
			if c.SectionID != current.SectionID && hasCommonTags(current, c) && !added[c.ID] {
				blocks.Related = append(blocks.Related, *c)
				added[c.ID] = true
			}
		}
	}
}

func buildSeriesBlocks(blocks *GeneratedBlocks, current *RenderedContent, allContent []*RenderedContent, maxItems int) {
	var seriesPosts []*RenderedContent
	for _, c := range allContent {
		if c.Series == current.Series {
			seriesPosts = append(seriesPosts, c)
		}
	}

	sort.Slice(seriesPosts, func(i, j int) bool {
		return seriesPosts[i].SeriesOrder < seriesPosts[j].SeriesOrder
	})

	currentIndex := -1
	for i, p := range seriesPosts {
		if p.ID == current.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return
	}

	if currentIndex > 0 {
		blocks.SeriesPrev = seriesPosts[currentIndex-1]
	}
	if currentIndex < len(seriesPosts)-1 {
		blocks.SeriesNext = seriesPosts[currentIndex+1]
	}

	if currentIndex < len(seriesPosts)-1 {
		for _, p := range seriesPosts[currentIndex+1:] {
			blocks.SeriesIndexForward = append(blocks.SeriesIndexForward, *p)
		}
	}

	if currentIndex > 0 {
		previousPosts := seriesPosts[:currentIndex]
		for i := len(previousPosts) - 1; i >= 0; i-- {
			blocks.SeriesIndexBackward = append(blocks.SeriesIndexBackward, *previousPosts[i])
		}
	}

	blocks.SeriesIndexForward = limitBlocks(blocks.SeriesIndexForward, maxItems)
	blocks.SeriesIndexBackward = limitBlocks(blocks.SeriesIndexBackward, maxItems)
}

func hasCommonTags(c1, c2 *RenderedContent) bool {
	for _, t1 := range c1.Tags {
		for _, t2 := range c2.Tags {
			if t1.ID == t2.ID {
				return true
			}
		}
	}
	return false
}

func limitBlocks(content []RenderedContent, max int) []RenderedContent {
	if len(content) > max {
		return content[:max]
	}
	return content
}
