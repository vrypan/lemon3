package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vrypan/lemon3/config"
	"github.com/vrypan/lemon3/fcclient"
	"github.com/vrypan/lemon3/ipfsclient"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Uploads file to ipfs, and creates a cast with lemon3 embeds",
	Run:   upload,
}

func upload(cmd *cobra.Command, args []string) {
	configFile := config.Load()
	if configFile == "" {
		fmt.Println("Please run \"lemon3 setup\" first.")
		return
	}

	if len(args) == 0 {
		cmd.Help()
		return
	}

	artwork, _ := cmd.Flags().GetString("artwork")
	if artwork == "" {
		fmt.Println("You need to provide an artwork file (jpeg, or png)")
		return
	}

	ipfsclient.Init(config.GetString("ipfs.hub"))
	fmt.Println()

	// Upload file
	fpath := args[0]
	cid, err := ipfsclient.AddFile(fpath)
	if err != nil {
		panic(err)
	}

	err = ipfsclient.PinCID(cid)
	if err != nil {
		fmt.Println("\n[!] Failed to pin data!")
		return
	}
	fmt.Printf("[+] %s pinned.\n", cid)

	// Upload artwork
	artworkCid, err := ipfsclient.AddFile(artwork)
	if err != nil {
		panic(err)
	}
	err = ipfsclient.PinCID(artworkCid)
	if err != nil {
		fmt.Println("\n[!] Failed to pin artwork!")
		return
	}
	fmt.Printf("[+] %s pinned.\n", artworkCid)

	mimeType, err := detectMimeType(fpath)
	fileSize, err := getFileSize(fpath)
	fileName := filepath.Base(fpath)
	fileTitle := fileName
	fileDescription := ""

	var s string
	if s, _ = cmd.Flags().GetString("title"); s != "" {
		fileTitle = s
	}
	if s, _ = cmd.Flags().GetString("name"); s != "" {
		fileName = s
	}
	if s, _ = cmd.Flags().GetString("mime"); s != "" {
		mimeType = s
	}

	fileDescription, _ = cmd.Flags().GetString("description")
	if strings.HasPrefix(fileDescription, "@") {
		source := strings.TrimPrefix(fileDescription, "@")

		var data []byte
		var err error

		if source == "-" {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading description from stdin: %v\n", err)
				os.Exit(1)
			}
		} else {
			data, err = os.ReadFile(source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading description file: %v\n", err)
				os.Exit(1)
			}
		}
		fileDescription = strings.TrimSpace(string(data))
	}

	data := map[string]any{
		"title":       fileTitle,
		"description": fileDescription,
		"type":        mimeType,
		"filename":    fileName,
		"size":        fileSize,
		"enclosed":    map[string]string{"/": cid},
		"artwork":     map[string]string{"/": artworkCid},
	}
	dagCid, err := ipfsclient.DagPut(data)
	if err != nil {
		fmt.Println("\n[!] Failed to upload metadate!")
		return
	}
	fmt.Printf("[^] Metadata cid=%s\n", dagCid)

	err = ipfsclient.PinCID(dagCid)
	if err != nil {
		fmt.Println("\n[!] Failed to pin metadata!")
		return
	}
	err = ipfsclient.ProvideCIDRecursive(dagCid)
	if err != nil {
		fmt.Println("\n[!] Failed to announce metadata.")
		return
	}
	fmt.Printf("[+] %s pinned.\n", dagCid)

	if WaitForCID(dagCid, 5, 10) != nil {
		fmt.Println("Exiting")
		return
	}

	hubConf := fcclient.HubConfig{
		Host: config.GetString("farcaster.node.address"),
		Ssl:  config.GetString("farcaster.node.ssl") == "true",
		Key:  config.GetString("farcaster.node.apikey"),
	}
	fmt.Println(hubConf)
	username := config.GetString("farcaster.account.fname")
	userkey := config.GetString("farcaster.account.appkey")

	castText, err := cmd.Flags().GetString("cast")
	castHash := fcclient.Cast(hubConf, username, userkey, castText, dagCid)
	fmt.Printf("[^] Cast posted: @%s/0x%s\n", username, castHash)

	fmt.Printf("\nView cast: https://farcaster.xyz/%s/0x%s\n", username, castHash)

}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().String("title", "", "Title")
	uploadCmd.Flags().String("name", "", "Filename (override original filename)")
	uploadCmd.Flags().String("mime", "", "mime/type (override automatic mime/type detection)")
	uploadCmd.Flags().String("description", "", "Description. @file will read the text from file, @- will read the text from stdin.")
	uploadCmd.Flags().String("artwork", "", "Path to artwork image.")
	uploadCmd.Flags().String("cast", "Uploaded with lemon3", "Cast text")
}

func detectMimeType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the first 512 bytes (used for detection)
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Detect content type
	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}

func getFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func WaitForCID(cid string, interval int, attempts int) error {
	url := fmt.Sprintf("https://ipfs.io/ipfs/%s", cid)
	spinner := []rune{'|', '/', '-', '\\'}
	for attempt := 1; attempt <= attempts; attempt++ {
		// Show spinner animation for ~5 seconds
		for i := 0; i < 20; i++ {
			fmt.Printf("\r[%c] Checking for %s on ipfs.io (attempt %d/10)", spinner[(attempt*20+i)%len(spinner)], cid, attempt)
			time.Sleep(250 * time.Millisecond)
		}

		// Perform HEAD request
		resp, err := http.Head(url)
		if err != nil {
			fmt.Printf("\r[!] Attempt %d failed: %v\n", attempt, err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("\r[✓] CID %s is now available on ipfs.io            \n", cid)
				return nil
			}
		}
	}
	fmt.Printf("\r[×] CID not available on ipfs.io                                                \n")
	return fmt.Errorf("CID %s not available after %d attempts", cid, attempts)
}
