package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upLyricsV2Shape, downLyricsV2Shape)
}

// upLyricsV2Shape reshapes the JSON stored in media_file.lyrics from the
// legacy v1 [{line: [...]}, ...] structure to the canonical v2 shape
// [{cueLine: [...], synced, ...}]. This is a synchronous in-place rewrite
// performed during migration. After the rewrite a full rescan is forced so
// any external Lyricsfile YAML or ELRC sidecars are picked up to populate
// the now-available v2 fields (per-word cues, agents, kind).
func upLyricsV2Shape(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `select id, lyrics from media_file where lyrics is not null and lyrics != '' and lyrics != '[]'`)
	if err != nil {
		return fmt.Errorf("scan media_file.lyrics: %w", err)
	}

	type updatePair struct {
		id      string
		payload string
	}
	var updates []updatePair

	for rows.Next() {
		var id, lyricsJSON string
		if err := rows.Scan(&id, &lyricsJSON); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan row: %w", err)
		}
		reshaped, err := reshapeLyricsV1ToV2(lyricsJSON)
		if err != nil {
			// Skip malformed rows; the forced rescan below will attempt to
			// repopulate them from source.
			continue
		}
		if reshaped == "" {
			continue
		}
		updates = append(updates, updatePair{id: id, payload: reshaped})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("scan media_file.lyrics: %w", err)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `update media_file set lyrics = ? where id = ?`)
	if err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.payload, u.id); err != nil {
			return fmt.Errorf("update lyrics for id=%s: %w", u.id, err)
		}
	}

	notice(tx, "Reshaped existing lyrics to v2; a full rescan will run to populate enhanced lyric data from sources")
	return forceFullRescan(tx)
}

func downLyricsV2Shape(_ context.Context, _ *sql.Tx) error {
	return nil
}

// reshapeLyricsV1ToV2 takes a JSON document that was historically a v1
// LyricList ([{lang, line: [{start, value}], ...}]) and rewrites it as the
// canonical v2 shape ([{lang, cueLine: [{index, start, end, value}], ...}]).
//
// All structures are local to the migration so that future changes to the
// model package do not silently change the migration's wire interpretation.
func reshapeLyricsV1ToV2(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	var src []v1Lyric
	if err := json.Unmarshal([]byte(input), &src); err != nil {
		return "", err
	}

	out := make([]v2Lyric, 0, len(src))
	for _, l := range src {
		// Skip rows that already look like the new shape.
		if len(l.Line) == 0 && len(l.CueLine) > 0 {
			out = append(out, v2Lyric{
				DisplayArtist: l.DisplayArtist,
				DisplayTitle:  l.DisplayTitle,
				Lang:          l.Lang,
				Offset:        l.Offset,
				Synced:        l.Synced,
				CueLine:       l.CueLine,
			})
			continue
		}

		cueLines := make([]v2CueLine, len(l.Line))
		for i, ln := range l.Line {
			cueLines[i] = v2CueLine{
				Index: i,
				Start: ln.Start,
				Value: ln.Value,
			}
		}
		// Infer end-of-line from the next line's start.
		for i := 0; i < len(cueLines)-1; i++ {
			if cueLines[i].End == nil && cueLines[i+1].Start != nil {
				cueLines[i].End = cueLines[i+1].Start
			}
		}

		out = append(out, v2Lyric{
			DisplayArtist: l.DisplayArtist,
			DisplayTitle:  l.DisplayTitle,
			Lang:          l.Lang,
			Offset:        l.Offset,
			Synced:        l.Synced,
			CueLine:       cueLines,
		})
	}

	bs, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// v1Lyric is the historical wire shape. CueLine is included so that rows that
// were already migrated by a partial run are passed through unchanged.
type v1Lyric struct {
	DisplayArtist string      `json:"displayArtist,omitempty"`
	DisplayTitle  string      `json:"displayTitle,omitempty"`
	Lang          string      `json:"lang"`
	Line          []v1Line    `json:"line,omitempty"`
	CueLine       []v2CueLine `json:"cueLine,omitempty"`
	Offset        *int64      `json:"offset,omitempty"`
	Synced        bool        `json:"synced"`
}

type v1Line struct {
	Start *int64 `json:"start,omitempty"`
	Value string `json:"value"`
}

type v2Lyric struct {
	DisplayArtist string      `json:"displayArtist,omitempty"`
	DisplayTitle  string      `json:"displayTitle,omitempty"`
	Lang          string      `json:"lang"`
	Offset        *int64      `json:"offset,omitempty"`
	Synced        bool        `json:"synced"`
	CueLine       []v2CueLine `json:"cueLine,omitempty"`
}

type v2CueLine struct {
	Index int    `json:"index"`
	Start *int64 `json:"start,omitempty"`
	End   *int64 `json:"end,omitempty"`
	Value string `json:"value"`
}
