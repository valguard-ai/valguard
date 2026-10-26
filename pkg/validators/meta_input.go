//go:build !osspublic

package validators

import (
	"context"
	"fmt"
)

// corpusData holds the joined input/RAG corpus and derived lookup sets.
type corpusData struct {
	text     string
	numSet   map[string]struct{}
	tokenSet map[string]struct{}
}

func buildCorpusData(text string) corpusData {
	if text == "" {
		return corpusData{}
	}
	nums := extractNumbers(text)
	numSet := make(map[string]struct{}, len(nums))
	for _, n := range nums {
		numSet[n] = struct{}{}
	}
	return corpusData{
		text:     text,
		numSet:   numSet,
		tokenSet: tokenSet(text),
	}
}

// inputJSONObjectFrom returns parsed input JSON from RunMeta on ctx (OSS CLI path).
func inputJSONObjectFrom(ctx context.Context) (map[string]any, error) {
	meta := RunMetaFrom(ctx)
	if meta.InputText == "" {
		return nil, fmt.Errorf("no input text")
	}
	return parseJSONObject(meta.InputText)
}

// corpusDataFrom returns corpus text and derived number/token sets from RunMeta.
func corpusDataFrom(ctx context.Context) corpusData {
	return buildCorpusData(ragCorpus(RunMetaFrom(ctx)))
}
