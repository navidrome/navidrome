package playback

import (
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Queue struct {
	Index int
	Items model.MediaFiles
}

func NewQueue() *Queue {
	return &Queue{
		Index: -1,
		Items: model.MediaFiles{},
	}
}

func (pd *Queue) String() string {
	filenames := ""
	for _, item := range pd.Items {
		filenames += item.Path + " "
	}
	return fmt.Sprintf("#Items: %d, idx: %d, files: %s", len(pd.Items), pd.Index, filenames)
}

// returns the current mediafile or nil
func (pd *Queue) Current() *model.MediaFile {
	if pd.Index == -1 {
		return nil
	}
	if pd.Index >= len(pd.Items) {
		log.Error("internal error: current song index out of bounds", "idx", pd.Index, "length", len(pd.Items))
		return nil
	}

	return &pd.Items[pd.Index]
}

// returns the whole queue
func (pd *Queue) Get() model.MediaFiles {
	return pd.Items
}

// set is similar to a clear followed by a add, but will not change the currently playing track.
func (pd *Queue) Set(items model.MediaFiles) {
	pd.Clear()
	pd.Items = append(pd.Items, items...)
}

// adding mediafiles to the queue
func (pd *Queue) Add(items model.MediaFiles) {
	pd.Items = append(pd.Items, items...)
	if pd.Index == -1 && len(pd.Items) > 0 {
		pd.Index = 0
	}
}

// empties whole queue
func (pd *Queue) Clear() {
	pd.Index = -1
	pd.Items = nil
}

// idx Zero-based index of the song to skip to or remove.
func (pd *Queue) Remove(idx int) {}

func (pd *Queue) Shuffle() {}

// Sets the index to a new, valid value inside the Items. Values lower than zero are going to be zero,
// values above will be limited by number of items.
func (pd *Queue) SetIndex(idx int) {
	pd.Index = max(0, min(idx, len(pd.Items)-1))
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
