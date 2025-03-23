package ipfsServer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestIPFSServer(t *testing.T) {
	// Clean up test files and directory both before and after test
	cleanup := func() {
		if err := os.RemoveAll("./test-repo"); err != nil && !os.IsNotExist(err) {
			t.Logf("Cleanup error: %v", err)
		}
		if err := os.Remove("test.txt"); err != nil && !os.IsNotExist(err) {
			t.Logf("Cleanup error: %v", err)
		}
		if err := os.Remove("test2.txt"); err != nil && !os.IsNotExist(err) {
			t.Logf("Cleanup error: %v", err)
		}
	}

	cleanup()       // Clean before running test
	defer cleanup() // Ensure cleanup happens after test, even if it fails

	// Create a channel to capture errors from the test goroutine
	errCh := make(chan error, 1)

	// Create a channel to signal completion
	done := make(chan struct{}, 1)

	// Run the test with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run the test in a goroutine
	go func() {
		// Create and start IPFS server
		s, err := NewIpfsServer("./test-repo")
		if err != nil {
			errCh <- fmt.Errorf("failed to create IPFS server: %v", err)
			return
		}

		t.Log("Starting IPFS server...")
		if err := s.Start(); err != nil {
			errCh <- fmt.Errorf("failed to start IPFS server: %v", err)
			return
		}

		// Ensure server stops even if test fails
		defer func() {
			t.Log("Stopping IPFS server...")
			if err := s.Stop(); err != nil {
				t.Logf("Error stopping IPFS server: %v", err)
			}
			t.Log("IPFS server stopped")
			done <- struct{}{}
		}()

		// Create test file
		testText := "Hello, World!"
		if err := os.WriteFile("test.txt", []byte(testText), 0644); err != nil {
			errCh <- fmt.Errorf("failed to write test file: %v", err)
			return
		}

		// Add file to IPFS
		t.Log("Adding file to IPFS...")
		cid, err := s.AddFile("./test.txt")
		if err != nil {
			errCh <- fmt.Errorf("failed to add file to IPFS: %v", err)
			return
		}
		t.Logf("File added with CID: %s", cid)

		// Get file from IPFS
		t.Log("Getting file from IPFS...")
		if err := s.GetFile(cid, "./test2.txt"); err != nil {
			errCh <- fmt.Errorf("failed to get file from IPFS: %v", err)
			return
		}

		// Verify file contents
		contents1, err := os.ReadFile("./test.txt")
		if err != nil {
			errCh <- fmt.Errorf("failed to read original file: %v", err)
			return
		}

		contents2, err := os.ReadFile("./test2.txt")
		if err != nil {
			errCh <- fmt.Errorf("failed to read retrieved file: %v", err)
			return
		}

		if string(contents1) != string(contents2) {
			errCh <- fmt.Errorf("expected the file contents to be the same, but got '%s' and '%s'", string(contents1), string(contents2))
			return
		}

		t.Log("Test completed successfully")
	}()

	// Wait for either test completion, error, or timeout
	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-done:
		t.Log("Test finished normally")
	case <-ctx.Done():
		t.Fatal("Test timed out after 30 seconds")
	}
}
