package playback

import (
	"fmt"
	"math/rand"

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
	for idx, item := range pd.Items {
		filenames += fmt.Sprint(idx) + ":" + item.Path + " "
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

func (pd *Queue) Size() int {
	return len(pd.Items)
}

func (pd *Queue) IsEmpty() bool {
	return len(pd.Items) < 1
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
func (pd *Queue) Remove(idx int) {
	current := pd.Current()
	backupID := ""
	if current != nil {
		backupID = current.ID
	}

	pd.Items = append(pd.Items[:idx], pd.Items[idx+1:]...)

	var err error
	pd.Index, err = pd.getMediaFileIndexByID(backupID)
	if err != nil {
		// we seem to have deleted the current id, setting to default:
		pd.Index = -1
	}
}

func (pd *Queue) Shuffle() {
	current := pd.Current()
	backupID := ""
	if current != nil {
		backupID = current.ID
	}

	rand.Shuffle(len(pd.Items), func(i, j int) { pd.Items[i], pd.Items[j] = pd.Items[j], pd.Items[i] })

	var err error
	pd.Index, err = pd.getMediaFileIndexByID(backupID)
	if err != nil {
		log.Error("Could not find ID while shuffling: " + backupID)
	}
}

func (pd *Queue) getMediaFileIndexByID(id string) (int, error) {
	for idx, item := range pd.Items {
		if item.ID == id {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("ID not found in playlist: " + id)
}

// Sets the index to a new, valid value inside the Items. Values lower than zero are going to be zero,
// values above will be limited by number of items.
func (pd *Queue) SetIndex(idx int) {
	pd.Index = max(0, min(idx, len(pd.Items)-1))
}

// Are we at the last track?
func (pd *Queue) IsAtLastElement() bool {
	return (pd.Index + 1) >= len(pd.Items)
}

// Goto next index
func (pd *Queue) IncreaseIndex() {
	if !pd.IsAtLastElement() {
		pd.SetIndex(pd.Index + 1)
	}
}
