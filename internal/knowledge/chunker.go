package knowledge

import (
	"fmt"
	"strings"
)

type Chunker struct {
	maxChunkChars int
	overlapChars  int
}

// NewChunker 创建文本分块器，并保证滑动窗口参数合法。
func NewChunker(maxChunkChars, overlapChars int) (*Chunker, error) {
	if maxChunkChars <= 0 {
		return nil, fmt.Errorf(
			"max chunk chars must be greater than zero: %d",
			maxChunkChars,
		)
	}

	if overlapChars < 0 {
		return nil, fmt.Errorf(
			"overlap chars must not be negative: %d",
			overlapChars,
		)
	}

	if overlapChars >= maxChunkChars {
		return nil, fmt.Errorf(
			"overlap chars must be smaller than max chunk chars: overlap=%d max=%d",
			overlapChars,
			maxChunkChars,
		)
	}

	return &Chunker{
		maxChunkChars: maxChunkChars,
		overlapChars:  overlapChars,
	}, nil
}

// Chunk 把文本切成多个固定长度且相邻重叠的片段。
func (c *Chunker) Chunk(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	runes := []rune(text)
	step := c.maxChunkChars - c.overlapChars

	var chunks []string

	for start := 0; start < len(runes); start += step {
		end := start + c.maxChunkChars
		if end > len(runes) {
			end = len(runes)
		}

		chunks = append(chunks, string(runes[start:end]))

		if end == len(runes) {
			break
		}
	}

	return chunks
}
