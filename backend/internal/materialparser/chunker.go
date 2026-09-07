package materialparser

import (
	"crypto/sha256"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"whatsnext/backend/internal/model"
)

type Chunker struct {
	TargetRunes  int
	OverlapRunes int
}

func NewChunker(target, overlap int) *Chunker {
	return &Chunker{TargetRunes: target, OverlapRunes: overlap}
}
func (c *Chunker) Build(material model.LearningMaterial, units []SourceUnit) []model.MaterialChunk {
	chunks := make([]model.MaterialChunk, 0)
	var buffered []SourceUnit
	count := 0
	flush := func() {
		if len(buffered) == 0 {
			return
		}
		texts := make([]string, 0, len(buffered))
		for _, unit := range buffered {
			texts = append(texts, unit.Text)
		}
		chunks = append(chunks, newChunk(material, len(chunks), strings.Join(texts, "\n"), buffered[0].SourceType, buffered[0].Position, buffered[len(buffered)-1].Position))
		buffered = nil
		count = 0
	}
	for _, unit := range units {
		runes := []rune(unit.Text)
		if len(runes) > c.TargetRunes {
			flush()
			step := c.TargetRunes - c.OverlapRunes
			if step <= 0 {
				step = c.TargetRunes
			}
			for start := 0; start < len(runes); start += step {
				end := start + c.TargetRunes
				if end > len(runes) {
					end = len(runes)
				}
				chunks = append(chunks, newChunk(material, len(chunks), string(runes[start:end]), unit.SourceType, unit.Position, unit.Position))
				if end == len(runes) {
					break
				}
			}
			continue
		}
		if len(buffered) > 0 && (count+len(runes) > c.TargetRunes || buffered[0].SourceType != unit.SourceType) {
			flush()
		}
		buffered = append(buffered, unit)
		count += len(runes)
	}
	flush()
	return chunks
}
func newChunk(material model.LearningMaterial, index int, content, sourceType string, start, end int) model.MaterialChunk {
	sum := sha256.Sum256([]byte(content))
	return model.MaterialChunk{ID: newChunkID(), UserID: material.UserID, LearningSpaceID: material.LearningSpaceID, MaterialID: material.ID, ChunkIndex: index, Content: content, SourceType: sourceType, SourceStart: &start, SourceEnd: &end, CharCount: utf8.RuneCountInString(content), TokenEstimate: estimateTokens(content), ContentHash: sum[:]}
}
func estimateTokens(content string) int {
	han, other := 0, 0
	for _, value := range content {
		if unicode.Is(unicode.Han, value) {
			han++
		} else if !unicode.IsSpace(value) {
			other++
		}
	}
	result := han + int(math.Ceil(float64(other)/4))
	if result < 1 {
		return 1
	}
	return result
}
func newChunkID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
