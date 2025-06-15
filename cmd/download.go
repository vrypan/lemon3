package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vrypan/lemon3/config"
	"github.com/vrypan/lemon3/fcclient"
	"github.com/vrypan/lemon3/ipfsclient"
)

var downloadCmd = &cobra.Command{
	Use:   "download <cast>",
	Short: "Download file from a lemon3+ipfs:// cast",
	Long: `Download the lemon3-enclosed file contained in a cast.

The cast must be in the format @user/<hash>, for example
lemon3 download @vrypan.eth/0xcd3141a47b98685c292b55c44f932e221753e51b

Be carefule, you must provide the full hash, not the shortened version
used in Farcaster URLs.`,
	Run: download,
}

func download(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: lemon3 download @user/<hash>")
		return
	}

	configFile := config.Load()
	if configFile == "" {
		fmt.Println("Please run \"lemon3 setup\" first.")
		return
	}

	castID := args[0]
	parts := strings.Split(castID, "/")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "@") {
		fmt.Println("Invalid cast format. Use @user/<hash>")
		return
	}
	username := strings.TrimPrefix(parts[0], "@")
	hash := parts[1]

	hubConf := fcclient.HubConfig{
		Host: config.GetString("farcaster.node"),
		Ssl:  config.GetString("farcaster.ssl") == "true",
	}

	embeds, err := fcclient.CastGetEmbedUrls(hubConf, username, hash)
	if err != nil {
		fmt.Printf("[!] Failed to fetch cast: %v\n", err)
		return
	}

	var cid string
	for _, e := range embeds {
		if strings.HasPrefix(e, "lemon3+ipfs://") {
			cid = strings.TrimPrefix(e, "lemon3+ipfs://")
			break
		}
	}

	if cid == "" {
		fmt.Println("[!] Failed to extract CID from embeddeds.")
		return
	}

	ipfsclient.Init(config.GetString("ipfs.hub"))

	// Fetch DAG metadata
	metadata, err := ipfsclient.DagGet(cid)
	if err != nil {
		fmt.Printf("[!] Failed to fetch DAG: %v\n", err)
		return
	}

	enclosedField, ok := metadata["enclosed"]
	if !ok {
		fmt.Println("[!] DAG does not contain 'enclosed' field.")
		return
	}

	enclosedMap, ok := enclosedField.(map[string]any)
	if !ok {
		fmt.Println("[!] 'enclosed' field is not the expected type.")
		return
	}

	enclosed, ok := enclosedMap["/"].(string)
	if !ok {
		fmt.Println("[!] DAG does not contain valid 'enclosed' field.")
		return
	}

	filename, ok := metadata["filename"].(string)
	if !ok || filename == "" {
		filename = enclosed // fallback
	}

	fmt.Printf("[↓] Downloading %s from %s...\n", filename, enclosed)
	size, _ := metadata["size"].(float64) // JSON uses float64 for numbers
	err = downloadFromIPFS(enclosed, filename, int64(size))
	if err != nil {
		fmt.Printf("[!] Failed to download file: %v\n", err)
		return
	}

	fmt.Printf("\r[✓] Saved as %s\n", filename)
}

func downloadFromIPFS(cid string, outFile string, size int64) error {
	url := fmt.Sprintf("https://ipfs.io/ipfs/%s", cid)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch CID %s: HTTP %d", cid, resp.StatusCode)
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

func init() {
	rootCmd.AddCommand(downloadCmd)
}
