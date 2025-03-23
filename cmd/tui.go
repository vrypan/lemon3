/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vrypan/lemon3/config"
	"github.com/vrypan/lemon3/fctools"
	"github.com/vrypan/lemon3/ipfsServer"
	db "github.com/vrypan/lemon3/localdb"
	"github.com/vrypan/lemon3/ui"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"start", "tui"},
	Short:   "Run the terminal user interface",
	Run:     tui,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func tui(cmd *cobra.Command, args []string) {
	db.Open()
	defer db.Close()

	hub := fctools.NewFarcasterHub(config.GetString("hub.address")+":"+config.GetString("hub.port"), config.GetBool("hub.ssl"))
	if hub == nil {
		fmt.Println("Error connecting to Farcaster hub.")
		fmt.Println("Make sure the hub you are using allows gRPC streaming.")
		fmt.Println("Use `lemon3 config ls` to view your current configuration.")
		os.Exit(1)
	}
	repoPath := flag.String("repo", "", "Path to IPFS repository")
	checkConnection := flag.Bool("check-connection", false, "Check IPFS network connectivity")
	flag.Parse()

	// Set default repository path if not provided
	if *repoPath == "" {
		currentUser, err := user.Current()
		if err != nil {
			fmt.Printf("Error getting current user: %s\n", err)
			os.Exit(1)
		}
		*repoPath = filepath.Join(currentUser.HomeDir, ".lemon3", "ipfs-repo")
	}

	// Ensure repository directory exists
	err := os.MkdirAll(*repoPath, 0755)
	if err != nil {
		fmt.Printf("Error creating repository directory: %s\n", err)
		os.Exit(1)
	}

	// Print startup message
	fmt.Printf("Using IPFS repository at: %s\n", *repoPath)

	// Initialize IPFS server
	ipfs := ipfsServer.NewIpfsLemon(*repoPath)
	err = ipfs.Start()
	if err != nil {
		fmt.Printf("Error starting IPFS server: %s\n", err)
		os.Exit(1)
	}

	// If check-connection flag is set, check and report on connection
	if *checkConnection {
		fmt.Println("Checking IPFS network connectivity...")
		fmt.Println("Waiting for connections (15s)...")

		// Wait longer for connections to establish
		time.Sleep(15 * time.Second)

		// Check for peers
		peers, err := ipfs.GetPeers(context.Background())
		if err != nil {
			fmt.Printf("Error checking peers: %s\n", err)
			os.Exit(1)
		}

		if len(peers) == 0 {
			fmt.Println("Not connected to any IPFS peers.")
			fmt.Println("\nTips for troubleshooting:")
			fmt.Println("1. Check your internet connection")
			fmt.Println("2. Verify firewall settings")
			fmt.Println("3. Try running IPFS daemon separately for detailed logs: ipfs daemon")
			fmt.Println("4. If using custom network, verify bootstrap nodes")
		} else {
			fmt.Printf("Connected to %d IPFS peers!\n", len(peers))
			for i, peer := range peers {
				if i < 5 { // Only show first 5 peers
					fmt.Printf("- %s\n", peer)
				}
			}
			if len(peers) > 5 {
				fmt.Printf("- ...and %d more\n", len(peers)-5)
			}
		}

		// Stop IPFS server
		ipfs.Stop()
		return
	}

	// Start UI with our running IPFS server
	fid := uint64(config.GetInt("key.fid"))
	k := config.GetString("key.private")
	var privateKey []byte
	if k != "" {
		if privateKey, err = hex.DecodeString(strings.TrimPrefix(k, "0x")); err != nil {
			log.Fatalf("Private key error: %v\nUse --help to see options.", err)
		}
	}

	model := ui.MainModel{
		Ipfs:       ipfs,
		Hub:        hub,
		Fid:        fid,
		PrivateKey: privateKey,
	}

	// Load old casts from specific fnames
	for _, fname := range args {
		fid, err := hub.GetFidByUsername(fname)
		if err != nil {
			fmt.Printf("Error getting FID for %s: %s\n", fname, err)
			continue
		}
		model.UpdateWithOldCasts(fid)
	}

	p := tea.NewProgram(&model, tea.WithAltScreen())
	exitModel, err := p.Run()
	if err != nil {
		fmt.Println(err)
	}
	err = exitModel.(*ui.MainModel).Err
	if err != nil {
		fmt.Println(err)
		fmt.Println()
		fmt.Println("If you're experiencing hub issues, make sure")
		fmt.Println("you are using a hub that allows gRPC streaming.")
		fmt.Println("Most public hubs don't.")
	}
}
