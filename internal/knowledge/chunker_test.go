package knowledge

import (
	"reflect"
	"testing"
)

func TestNewChunkerRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		maxChunkChars int
		overlapChars  int
	}{
		{name: "non-positive max", maxChunkChars: 0, overlapChars: 0},
		{name: "negative overlap", maxChunkChars: 5, overlapChars: -1},
		{name: "overlap equals max", maxChunkChars: 5, overlapChars: 5},
		{name: "overlap exceeds max", maxChunkChars: 5, overlapChars: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chunker, err := NewChunker(tt.maxChunkChars, tt.overlapChars)
			if err == nil {
				t.Fatalf("NewChunker(%d, %d) error = nil, want error", tt.maxChunkChars, tt.overlapChars)
			}
			if chunker != nil {
				t.Fatalf("NewChunker(%d, %d) chunker = %#v, want nil", tt.maxChunkChars, tt.overlapChars, chunker)
			}
		})
	}
}

func TestChunkHandlesEmptyAndShortText(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(5, 2)
	if err != nil {
		t.Fatalf("NewChunker() error = %v", err)
	}

	if got := chunker.Chunk("  \n\t "); got != nil {
		t.Fatalf("Chunk(whitespace) = %#v, want nil", got)
	}

	want := []string{"你好"}
	if got := chunker.Chunk("你好"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Chunk(short text) = %#v, want %#v", got, want)
	}
}

func TestChunkUsesOverlappingRuneWindows(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(5, 2)
	if err != nil {
		t.Fatalf("NewChunker() error = %v", err)
	}

	want := []string{"甲乙丙丁戊", "丁戊己庚辛", "庚辛壬癸"}
	got := chunker.Chunk("甲乙丙丁戊己庚辛壬癸")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Chunk() = %#v, want %#v", got, want)
	}
}
