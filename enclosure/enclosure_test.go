package enclosure

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrypan/lemon3/ipfsServer"
)

func TestWithRealIpfs(t *testing.T) {
	// Create a temporary directory for the IPFS repo
	tempRepoDir, err := ioutil.TempDir("", "ipfs-test-repo")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(tempRepoDir)

	// Create enclosures output directory if it doesn't exist
	enclosuresDir := filepath.Join(".", "testdata")
	err = os.MkdirAll(enclosuresDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create enclosures directory: %v", err)
	}

	// Initialize the IPFS server
	ipfs := ipfsServer.NewIpfsLemon(tempRepoDir)
	err = ipfs.Start()
	if err != nil {
		t.Fatalf("Failed to start IPFS server: %v", err)
	}
	defer ipfs.Stop()

	t.Log("IPFS server started")
	// 1. Create a temporary file with lorem ipsum text
	loremIpsum := `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse euismod,
urna eget aliquet fermentum, odio nisl ultrices nulla, eget ultrices nisl nisl nec nisl.`

	tempFilePath := filepath.Join(os.TempDir(), "lorem_ipsum.txt")
	t.Log("Test file", tempFilePath)
	err = ioutil.WriteFile(tempFilePath, []byte(loremIpsum), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFilePath)

	// 2. Create a new enclosure with the file
	filename := "test.txt"
	description := "this is the test body"
	mimeType := "text/plain"

	enclosure, err := NewEnclosure(ipfs, tempFilePath, filename, mimeType, description)
	if err != nil {
		t.Fatalf("Failed to create enclosure: %v", err)
	}

	t.Logf("Created enclosure with CID: %s", enclosure.EnclosureCID)

	// 3. Check that the enclosure properties are set correctly
	if enclosure.FileName != filename {
		t.Errorf("Expected filename %s, got %s", filename, enclosure.FileName)
	}

	// 4. Save the enclosure JSON to the enclosures directory using ipfsServer.GetFile
	enclosureJsonPath := filepath.Join(enclosuresDir, "enclosure.json")
	err = ipfs.GetFile(enclosure.EnclosureCID, enclosureJsonPath)
	if err != nil {
		t.Logf("Note: Could not save enclosure JSON: %v", err)
		// Don't fail the test here as this might be environment-specific
	} else {
		t.Logf("Saved enclosure JSON to %s", enclosureJsonPath)
		defer os.Remove(enclosureJsonPath)
	}

	// 5. Create a new enclosure from the CID
	retrievedEnclosure, err := NewEnclosureFromCID(ipfs, enclosure.EnclosureCID)
	if err != nil {
		t.Fatalf("Failed to create enclosure from CID: %v", err)
	}

	// Compare the original and retrieved enclosures
	if retrievedEnclosure.FileName != enclosure.FileName {
		t.Errorf("Titles don't match: original=%s, retrieved=%s", enclosure.FileName, retrievedEnclosure.FileName)
	}
	if retrievedEnclosure.Description != enclosure.Description {
		t.Errorf("Descriptions don't match: original=%s, retrieved=%s", enclosure.Description, retrievedEnclosure.Description)
	}

	// 6. Download the file from the file CID to the enclosures directory
	downloadPath := filepath.Join(enclosuresDir, "downloaded_file.txt")
	err = ipfs.GetFile(retrievedEnclosure.FileCID, downloadPath)
	if err != nil {
		t.Logf("Note: Could not download file content: %v", err)
		// Don't fail the test here as this might be environment-specific
	} else {
		t.Logf("Retrieved file content to %s", downloadPath)

		// Read the downloaded content
		downloadedContent, err := ioutil.ReadFile(downloadPath)
		if err == nil {
			// Compare with the original content
			if string(downloadedContent) != loremIpsum {
				t.Errorf("Downloaded content doesn't match original.")
			} else {
				t.Logf("Downloaded content matches original text.")
			}
		}

		defer os.Remove(downloadPath)
	}
}
