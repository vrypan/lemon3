package ipfsclient

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type AddResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"` // this is the CID
	Size string `json:"Size"`
}
type ProgressReader struct {
	io.Reader
	Total      int64
	ReadBytes  int64
	Callback   func(percent float64)
	lastUpdate time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.ReadBytes += int64(n)

	// Throttle updates to ~100ms
	now := time.Now()
	if now.Sub(pr.lastUpdate) > 100*time.Millisecond || err == io.EOF {
		pr.lastUpdate = now
		percent := (float64(pr.ReadBytes) / float64(pr.Total)) * 100
		pr.Callback(percent)
	}

	return n, err
}

func AddFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return "", err
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer file.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		progressReader := &ProgressReader{
			Reader:   file,
			Total:    stat.Size(),
			Callback: func(percent float64) { fmt.Printf("\r[^] Uploading %s: %.1f%%", filePath, percent) },
		}

		if _, err := io.Copy(part, progressReader); err != nil {
			pw.CloseWithError(err)
			return
		}
		fmt.Printf(" ")

		writer.Close()
	}()

	req, err := http.NewRequest("POST", kuboAPI+"/add", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %s", string(rb))
	}

	var result AddResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	fmt.Printf(" (cid=%s)\n", result.Hash)
	return result.Hash, nil
}
