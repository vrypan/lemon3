package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Run:   downloadFeed,
}

func downloadFeed(cmd *cobra.Command, args []string) {
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
	status_casts := []*lemon3libs.L3Cast{}
	for _, cast := range casts {
		l3cast, err := lemon3libs.FromPbMessage(cast)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if l3cast == nil {
			continue
		}
		if l3cast.Hash == lastCastHash {
			return
		}
		l3cast.Fname = username[1:]
		status_casts = append(status_casts, l3cast)

		enclosed := l3cast.Lemon3Data.Enclosed["/"]
		filename := l3cast.Lemon3Data.Filename

		fmt.Printf("[↓] Downloading %s from %s...\n", filename, enclosed)
		err = ipfsclient.CatCIDToFile(enclosed, filepath.Join(downloadPath, filename), l3cast.Lemon3Data.Size)
		if err != nil {
			fmt.Printf("[!] Failed to download file: %v\n", err)
			return
		}

		// fmt.Printf("[✓] Saved %s as %s               \n\n", cid, filename)
		fmt.Println()

	}

	status["casts"] = status_casts
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
