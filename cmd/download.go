package cmd

import (
	"fmt"
	"strings"

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
	err = ipfsclient.CatCIDToFile(enclosed, filename, int64(size))
	if err != nil {
		fmt.Printf("[!] Failed to download file: %v\n", err)
		return
	}

	fmt.Printf("\r[✓] Saved as %s\n", filename)
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
