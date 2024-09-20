package writer

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
)

type Update struct {
	Range   hcl.Range
	Content []byte
}

type Updates []Update

func (u Updates) Len() int {
	return len(u)
}

func (u Updates) Less(i int, j int) bool {
	return u[i].Range.Start.Byte < u[j].Range.Start.Byte
}

func (u Updates) Swap(i int, j int) {
	u[i], u[j] = u[j], u[i]
}

// UpdateContent update the original content with a series of updates.
// Each update shall has no overlap range with others, and the range has
// to be within the original content.
func UpdateContent(b []byte, updates Updates) ([]byte, error) {
	var nb []byte
	sort.Sort(updates)
	var startOffset int
	for i, update := range updates {
		if i != len(updates)-1 {
			nextUpdate := updates[i+1]
			if update.Range.Overlaps(nextUpdate.Range) {
				return nil, fmt.Errorf("overlapping ranges of updates found: %s vs %s", update.Range, nextUpdate.Range)
			}
		}
		if update.Range.End.Byte > len(b) {
			return nil, fmt.Errorf("update exceeded the raw content length: %s", update.Range)
		}
		nb = append(nb, b[startOffset:update.Range.Start.Byte]...)
		nb = append(nb, update.Content...)
		startOffset = update.Range.End.Byte
	}
	nb = append(nb, b[startOffset:]...)
	return nb, nil
}
