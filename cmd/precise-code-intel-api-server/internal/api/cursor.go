package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/bundles"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/db"
)

type Cursor struct {
	Phase                  string                // common
	DumpID                 int                   // common
	Path                   string                // same-dump/definition-monikers
	Line                   int                   // same-dump
	Character              int                   // same-dump
	Monikers               []bundles.MonikerData // same-dump/definition-monikers
	SkipResults            int                   // same-dump/definition-monikers
	Identifier             string                // same-repo/remote-repo
	Scheme                 string                // same-repo/remote-repo
	Name                   string                // same-repo/remote-repo
	Version                string                // same-repo/remote-repo
	DumpIDs                []int                 // same-repo/remote-repo
	TotalDumpsWhenBatching int                   // same-repo/remote-repo
	SkipDumpsWhenBatching  int                   // same-repo/remote-repo
	SkipDumpsInBatch       int                   // same-repo/remote-repo
	SkipResultsInDump      int                   // same-repo/remote-repo
}

func decodeCursor(rawEncoded string) (cursor Cursor, err error) {
	raw, err := base64.RawURLEncoding.DecodeString(rawEncoded)
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(raw), &cursor)
	return
}

func EncodeCursor(cursor Cursor) string {
	rawEncoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(rawEncoded)
}

// TODO - pass values separately instead
func DecodeCursorFromRequest(query url.Values, db db.DB, bundleManagerClient bundles.BundleManagerClient) (Cursor, error) {
	if encoded := query.Get("cursor"); encoded != "" {
		cursor, err := decodeCursor(encoded)
		if err != nil {
			return Cursor{}, err
		}

		return cursor, nil
	}

	file := query.Get("path")
	line, _ := strconv.Atoi(query.Get("line"))
	character, _ := strconv.Atoi(query.Get("character"))
	uploadID, _ := strconv.Atoi(query.Get("uploadId"))

	dump, exists, err := db.GetDumpByID(context.Background(), uploadID)
	if err != nil {
		return Cursor{}, err
	}
	if !exists {
		return Cursor{}, ErrMissingDump
	}

	pathInBundle := strings.TrimPrefix(file, dump.Root)
	bundleClient := bundleManagerClient.BundleClient(dump.ID)

	rangeMonikers, err := bundleClient.MonikersByPosition(context.Background(), pathInBundle, line, character)
	if err != nil {
		return Cursor{}, err
	}

	var flattened []bundles.MonikerData
	for _, monikers := range rangeMonikers {
		flattened = append(flattened, monikers...)
	}

	return Cursor{
		Phase:       "same-dump",
		DumpID:      dump.ID,
		Path:        pathInBundle,
		Line:        line,
		Character:   character,
		Monikers:    flattened,
		SkipResults: 0,
	}, nil
}