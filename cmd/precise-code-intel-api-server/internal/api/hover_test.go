package api

import (
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/bundles"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/db"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/mocks"
)

func TestHover(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	mockBundleClient := mocks.NewMockBundleClient()

	setMockDBGetDumpByID(t, mockDB, map[int]db.Dump{42: testDump1})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{42: mockBundleClient})
	setMockBundleClientHover(t, mockBundleClient, "main.go", 10, 50, "text", testRange1, true)

	api := &codeIntelAPI{
		db:                  mockDB,
		bundleManagerClient: mockBundleManagerClient,
	}

	text, r, exists, err := api.Hover("sub1/main.go", 10, 50, 42)
	if err != nil {
		t.Fatalf("expected error getting hover text: %s", err)
	}
	if !exists {
		t.Fatalf("expected hover text to exist.")
	}

	if text != "text" {
		t.Errorf("unexpected text. want=%v have=%v", "text", text)
	}
	if !reflect.DeepEqual(r, testRange1) {
		t.Errorf("unexpected range. want=%v have=%v", testRange1, r)
	}
}

func TestHoverUnknownDump(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	setMockDBGetDumpByID(t, mockDB, nil)

	api := &codeIntelAPI{
		db:                  mockDB,
		bundleManagerClient: mockBundleManagerClient,
	}

	if _, _, _, err := api.Hover("sub1/main.go", 10, 50, 42); err != ErrMissingDump {
		t.Errorf("unexpected error getting hover text. want=%v have=%v", ErrMissingDump, err)
	}
}

func TestHoverRemoteDefinitionHoverText(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	mockBundleClient1 := mocks.NewMockBundleClient()
	mockBundleClient2 := mocks.NewMockBundleClient()

	setMockDBGetDumpByID(t, mockDB, map[int]db.Dump{42: testDump1, 50: testDump2})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{42: mockBundleClient1, 50: mockBundleClient2})
	setMockBundleClientHover(t, mockBundleClient1, "main.go", 10, 50, "", bundles.Range{}, false)
	setMockBundleClientDefinitions(t, mockBundleClient1, "main.go", 10, 50, nil)
	setMockBundleClientMonikersByPosition(t, mockBundleClient1, "main.go", 10, 50, [][]bundles.MonikerData{{testMoniker1}})
	setMockBundleClientPackageInformation(t, mockBundleClient1, "main.go", "1234", testPackageInformation)
	setMockDBGetPackage(t, mockDB, "gomod", "leftpad", "0.1.0", testDump2, true)
	setMockBundleClientMonikerResults(t, mockBundleClient2, "definitions", "gomod", "pad", 0, 0, []bundles.Location{
		{DumpID: 50, Path: "foo.go", Range: testRange1},
		{DumpID: 50, Path: "bar.go", Range: testRange2},
		{DumpID: 50, Path: "baz.go", Range: testRange3},
	}, 15)
	setMockBundleClientHover(t, mockBundleClient2, "foo.go", 10, 50, "text", testRange4, true)

	api := &codeIntelAPI{
		db:                  mockDB,
		bundleManagerClient: mockBundleManagerClient,
	}

	text, r, exists, err := api.Hover("sub1/main.go", 10, 50, 42)
	if err != nil {
		t.Fatalf("expected error getting hover text: %s", err)
	}
	if !exists {
		t.Fatalf("expected hover text to exist.")
	}

	if text != "text" {
		t.Errorf("unexpected text. want=%v have=%v", "text", text)
	}
	if !reflect.DeepEqual(r, testRange4) {
		t.Errorf("unexpected range. want=%v have=%v", testRange4, r)
	}
}

func TestHoverUnknownDefinition(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockBundleManagerClient := mocks.NewMockBundleManagerClient()
	mockBundleClient := mocks.NewMockBundleClient()

	setMockDBGetDumpByID(t, mockDB, map[int]db.Dump{42: testDump1})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{42: mockBundleClient})
	setMockBundleClientHover(t, mockBundleClient, "main.go", 10, 50, "", bundles.Range{}, false)
	setMockBundleClientDefinitions(t, mockBundleClient, "main.go", 10, 50, nil)
	setMockBundleClientMonikersByPosition(t, mockBundleClient, "main.go", 10, 50, [][]bundles.MonikerData{{testMoniker1}})
	setMockBundleClientPackageInformation(t, mockBundleClient, "main.go", "1234", testPackageInformation)
	setMockDBGetPackage(t, mockDB, "gomod", "leftpad", "0.1.0", db.Dump{}, false)

	api := &codeIntelAPI{
		db:                  mockDB,
		bundleManagerClient: mockBundleManagerClient,
	}

	_, _, exists, err := api.Hover("sub1/main.go", 10, 50, 42)
	if err != nil {
		t.Errorf("unexpected error getting hover text: %s", err)
	}
	if exists {
		t.Errorf("unexpected hover text")
	}
}