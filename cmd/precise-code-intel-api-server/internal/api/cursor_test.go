package api

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/bundles"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/db"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/mocks"
)

func TestSerializationRoundTrip(t *testing.T) {
	c := Cursor{
		Phase:     "same-repo",
		DumpID:    42,
		Path:      "/foo/bar/baz.go",
		Line:      10,
		Character: 50,
		Monikers: []bundles.MonikerData{
			{Kind: "k1", Scheme: "s1", Identifier: "i1", PackageInformationID: "pid1"},
			{Kind: "k2", Scheme: "s2", Identifier: "i2", PackageInformationID: "pid2"},
			{Kind: "k3", Scheme: "s3", Identifier: "i3", PackageInformationID: "pid3"},
		},
		SkipResults:            1,
		Identifier:             "x",
		Scheme:                 "gomod",
		Name:                   "leftpad",
		Version:                "0.1.0",
		DumpIDs:                []int{1, 2, 3, 4, 5},
		TotalDumpsWhenBatching: 5,
		SkipDumpsWhenBatching:  4,
		SkipDumpsInBatch:       3,
		SkipResultsInDump:      2,
	}

	roundtripped, err := decodeCursor(EncodeCursor(c))
	if err != nil {
		t.Fatalf("unexpected error decoding cursor: %s", err)
	}

	if !reflect.DeepEqual(c, roundtripped) {
		t.Errorf("unexpected cursor. want=%v have=%v", c, roundtripped)
	}
}

func TestDecodeFreshCursorFromRequest(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	mockBundleClient := mocks.NewMockBundleClient()

	setMockDBGetDumpByID(t, mockDB, map[int]db.Dump{42: testDump1})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{42: mockBundleClient})
	setMockBundleClientMonikersByPosition(t, mockBundleClient, "main.go", 10, 20, [][]bundles.MonikerData{{testMoniker1}, {testMoniker2}})

	query := url.Values{
		"path":      []string{"sub1/main.go"},
		"line":      []string{"10"},
		"character": []string{"20"},
		"uploadId":  []string{"42"},
	}

	expectedCursor := Cursor{
		Phase:     "same-dump",
		DumpID:    42,
		Path:      "main.go",
		Line:      10,
		Character: 20,
		Monikers:  []bundles.MonikerData{testMoniker1, testMoniker2},
	}

	if cursor, err := DecodeCursorFromRequest(query, mockDB, mockBundleManagerClient); err != nil {
		t.Fatalf("unexpected error decoding cursor: %s", err)
	} else if !reflect.DeepEqual(cursor, expectedCursor) {
		t.Errorf("unexpected cursor. want=%v have=%v", expectedCursor, cursor)
	}
}

func TestDecodeFreshCursorFromRequestUnknownDump(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	setMockDBGetDumpByID(t, mockDB, nil)

	query := url.Values{
		"path":      []string{"sub1/main.go"},
		"line":      []string{"10"},
		"character": []string{"20"},
		"uploadId":  []string{"42"},
	}

	if _, err := DecodeCursorFromRequest(query, mockDB, mockBundleManagerClient); err != ErrMissingDump {
		t.Fatalf("unexpected error decoding cursor. want=%v have =%v", ErrMissingDump, err)
	}
}

func TestDecodeExistingCursorFromRequest(t *testing.T) {
	expectedCursor := Cursor{
		Phase:     "same-repo",
		DumpID:    42,
		Path:      "/foo/bar/baz.go",
		Line:      10,
		Character: 50,
		Monikers: []bundles.MonikerData{
			{Kind: "k1", Scheme: "s1", Identifier: "i1", PackageInformationID: "pid1"},
			{Kind: "k2", Scheme: "s2", Identifier: "i2", PackageInformationID: "pid2"},
			{Kind: "k3", Scheme: "s3", Identifier: "i3", PackageInformationID: "pid3"},
		},
		SkipResults:            1,
		Identifier:             "x",
		Scheme:                 "gomod",
		Name:                   "leftpad",
		Version:                "0.1.0",
		DumpIDs:                []int{1, 2, 3, 4, 5},
		TotalDumpsWhenBatching: 5,
		SkipDumpsWhenBatching:  4,
		SkipDumpsInBatch:       3,
		SkipResultsInDump:      2,
	}

	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()

	query := url.Values{
		"cursor": []string{EncodeCursor(expectedCursor)},
	}

	if cursor, err := DecodeCursorFromRequest(query, mockDB, mockBundleManagerClient); err != nil {
		t.Fatalf("unexpected error decoding cursor: %s", err)
	} else if !reflect.DeepEqual(cursor, expectedCursor) {
		t.Errorf("unexpected cursor. want=%v have=%v", expectedCursor, cursor)
	}
}