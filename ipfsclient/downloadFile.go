package ipfsclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func CatCIDToFile(cid, outFile string, size int64) error {
	resp, err := http.Post(kuboAPI+"/cat?arg="+url.QueryEscape(cid), "application/x-www-form-urlencoded", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cat failed: %s", string(rb))
	}

	file, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Spinner + progress
	done := make(chan struct{})
	progress := make(chan int64)

	go func() {
		spin := []rune{'|', '/', '-', '\\'}
		i := 0
		var downloaded int64
		for {
			select {
			case <-done:
				fmt.Printf("\r[✓] Downloaded %d / %d bytes (100.0%%)                         \n", size, size)
				return
			case downloaded = <-progress:
				percentage := (float64(downloaded) / float64(size)) * 100
				fmt.Printf("\r[%c] Downloading... %d / %d bytes (%.1f%%)", spin[i%len(spin)], downloaded, size, percentage)
				i++
			case <-time.After(100 * time.Millisecond):
				percentage := (float64(downloaded) / float64(size)) * 100
				fmt.Printf("\r[%c] Downloading... %d / %d bytes (%.1f%%)", spin[i%len(spin)], downloaded, size, percentage)
				i++
			}
		}
	}()

	// Track progress
	countingReader := &countReader{Reader: resp.Body, progress: progress}
	_, err = io.Copy(file, countingReader)
	close(done)
	return err
}

// countReader wraps an io.Reader and sends progress updates
type countReader struct {
	Reader   io.Reader
	read     int64
	progress chan<- int64
}

func (cr *countReader) Read(p []byte) (int, error) {
	n, err := cr.Reader.Read(p)
	cr.read += int64(n)
	cr.progress <- cr.read
	return n, err
}
