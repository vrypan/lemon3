package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vrypan/lemon3/config"
	"github.com/vrypan/lemon3/fcclient"
	"github.com/vrypan/lemon3/ipfsclient"
	"github.com/vrypan/lemon3/lemon3libs"
)

var download2Cmd = &cobra.Command{
	Use:   "downloadfeed <user>",
	Short: "Download lemon3 files shared by a user",
	Run:   download2,
}

func download2(cmd *cobra.Command, args []string) {
	configFile := config.Load()
	if configFile == "" {
		fmt.Println("Please run \"lemon3 setup\" first.")
		return
	}
	if len(args) == 0 {
		fmt.Println("Usage: lemon3 download @user")
		return
	}
	username := args[0]

	hubConf := fcclient.HubConfig{
		Host: config.GetString("farcaster.node.address"),
		Ssl:  config.GetString("farcaster.node.ssl") == "true",
		Key:  config.GetString("farcaster.node.apikey"),
	}
	fcclient.Init(hubConf)
	ipfsclient.Init(config.GetString("ipfs.hub"))

	downloadPath := filepath.Join(config.GetString("download.dir"), username[1:])
	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		if err := os.MkdirAll(downloadPath, 0755); err != nil {
			fmt.Printf("[!] Failed to create download directory: %v\n", err)
			return
		}
	}

	statusFile := filepath.Join(downloadPath, ".lemon3")
	var lastCastHash string
	status := make(map[string]any)

	statusDataBytes, err := os.ReadFile(statusFile)
	if err == nil {
		if err := json.Unmarshal(statusDataBytes, &status); err != nil {
			fmt.Printf("[!] Failed to parse status file as JSON: %v\n", err)
			return
		}
		if hash, ok := status["last_hash"].(string); ok {
			lastCastHash = hash
		} else {
			fmt.Println("[!] Failed to get last_hash from status file.")
			return
		}
	}

	casts, err := fcclient.GetCastsByFname(username[1:], 100, true)
	if err != nil {
		fmt.Printf("[!] Failed to get casts: %v\n", err)
		os.Exit(1)
	}
	if len(casts) == 0 {
		fmt.Println("[!] No casts found for user.")
		return
	}

	status["last_hash"] = fmt.Sprintf("0x%x", casts[0].Hash)
	for _, cast := range casts {
		hash := fmt.Sprintf("0x%x", cast.Hash)
		if hash == lastCastHash && lastCastHash != "" {
			fmt.Println("Found last downloaded cast, stopping.")
			break
		}
		for _, e := range cast.Data.GetCastAddBody().GetEmbeds() {
			if strings.HasPrefix(e.GetUrl(), "lemon3+ipfs://") {
				cid := strings.TrimPrefix(e.GetUrl(), "lemon3+ipfs://")
				fmt.Printf("[✓] %s %s -> %s\n", tsToDate(cast.Data.GetTimestamp()), hash, cid)

				meta, err := lemon3libs.FromCid(cid)
				if err != nil {
					fmt.Printf("[!] %v\n", err)
					return
				}
				enclosed := meta.Enclosed["/"]
				filename := meta.Filename

				fmt.Printf("[↓] Downloading %s from %s...\n", filename, enclosed)
				err = ipfsclient.CatCIDToFile(enclosed, filepath.Join(downloadPath, filename), meta.Size)
				if err != nil {
					fmt.Printf("[!] Failed to download file: %v\n", err)
					return
				}

				fmt.Printf("\r[✓] Saved as %s\n\n", filename)
			}
		}
	}

	statusBytes, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Printf("[!] Failed to serialize status as JSON: %v\n", err)
		return
	}
	if err := os.WriteFile(statusFile, statusBytes, 0644); err != nil {
		fmt.Printf("[!] Failed to write status file: %v\n", err)
		return
	}
}

func init() {
	rootCmd.AddCommand(download2Cmd)
}

func tsToDate(ts uint32) string {
	return time.Unix(int64(ts)+fcclient.FARCASTER_EPOCH, 0).Format("2006-01-02 15:04:05")
}
