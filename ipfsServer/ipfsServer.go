package ipfsServer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	config "github.com/ipfs/kubo/config"
	core "github.com/ipfs/kubo/core"
	coreapi "github.com/ipfs/kubo/core/coreapi"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	loader "github.com/ipfs/kubo/plugin/loader"
	fsrepo "github.com/ipfs/kubo/repo/fsrepo"

	files "github.com/ipfs/boxo/files"
	path "github.com/ipfs/boxo/path"
)

// IpfsServer represents an IPFS server
type IpfsServer struct {
	mutex     sync.RWMutex
	repoPath  string
	ipfs      *core.IpfsNode
	coreAPI   coreiface.CoreAPI
	cmdCtx    context.Context
	cmdCancel context.CancelFunc
	running   bool
}

func NewIpfsLemon(repoPath string) *IpfsServer {
	return &IpfsServer{
		repoPath: repoPath,
		running:  false,
	}
}

func (m *IpfsServer) IsRunning() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.running
}

// SetupPlugins loads the preloaded plugins
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func (m *IpfsServer) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return nil
	}

	ctx := context.Background()

	// Load IPFS plugins
	plugins, err := loader.NewPluginLoader("")
	if err != nil {
		return fmt.Errorf("error loading plugins: %v", err)
	}
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %v", err)
	}
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error injecting plugins: %v", err)
	}

	// Initialize the IPFS repo if not already
	if !fsrepo.IsInitialized(m.repoPath) {
		cfg, err := config.Init(io.Discard, 2048)
		if err != nil {
			return fmt.Errorf("failed to initialize config: %v", err)
		}

		// Modify config for test environment
		cfg.Experimental.FilestoreEnabled = false
		cfg.Experimental.ShardingEnabled = false
		cfg.Swarm.DisableNatPortMap = false
		cfg.Routing.Type = config.NewOptionalString("auto")

		err = fsrepo.Init(m.repoPath, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize repo: %v", err)
		}
	}

	repo, err := fsrepo.Open(m.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %v", err)
	}

	nodeOptions := &core.BuildCfg{
		Online: true,
		Repo:   repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return fmt.Errorf("failed to create IPFS node: %v", err)
	}

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return fmt.Errorf("failed to create CoreAPI: %v", err)
	}

	m.ipfs = node
	m.coreAPI = api
	m.running = true

	// Initialize the core API
	m.coreAPI, err = coreapi.NewCoreAPI(m.ipfs)
	if err != nil {
		return fmt.Errorf("error creating core API: %s", err)
	}

	// Connect to bootstrap nodes in the background
	/*go func() {
		// Give the node a moment to initialize
		time.Sleep(2 * time.Second)

		//log.Println("Attempting to connect to bootstrap nodes...")
		ctx := context.Background()

		// Fixed list of common IPFS bootstrap peers
		bootstrapNodes := []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
			"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		}

		var connectedPeers int

		// Try to connect to each bootstrap node
		for _, addrStr := range bootstrapNodes {
			addr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				log.Printf("Invalid bootstrap address: %s, %s", addrStr, err)
				continue
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				log.Printf("Failed to parse peer info from address: %s, %s", addrStr, err)
				continue
			}

			err = m.ipfs.PeerHost.Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("Failed to connect to peer %s: %s", addrStr, err)
			} else {
				//log.Printf("Successfully connected to peer: %s", addrStr)
				connectedPeers++
			}
		}

		//log.Printf("Connected to %d bootstrap nodes", connectedPeers)
	}()
	*/

	return nil
}

// AddFile adds a file to IPFS and returns its CID
func (m *IpfsServer) AddFile(filePath string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if IPFS is running - don't call IsRunning() here as we already have the lock
	if !m.running {
		return "", fmt.Errorf("IPFS server is not running")
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create fileNode
	fileNode := files.NewReaderFile(file)

	// Add file to IPFS with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cidPath, err := m.coreAPI.Unixfs().Add(ctx, fileNode)
	if err != nil {
		return "", fmt.Errorf("failed to add file to IPFS: %v", err)
	}

	// Pin file with timeout
	pinCtx, pinCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pinCancel()

	err = m.coreAPI.Pin().Add(pinCtx, cidPath)
	if err != nil {
		return "", fmt.Errorf("failed to pin file: %v", err)
	}

	// Get CID string
	cid := cidPath.String()

	return cid, nil
}

// AddBytes adds byte array to IPFS and returns its CID
func (m *IpfsServer) AddBytes(content []byte) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if IPFS is running - don't call IsRunning() here as we already have the lock
	if !m.running {
		return "", fmt.Errorf("IPFS server is not running")
	}

	// Create contentNode from byte array
	contentNode := files.NewReaderFile(bytes.NewReader(content))

	// Add content to IPFS with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cidPath, err := m.coreAPI.Unixfs().Add(ctx, contentNode)
	if err != nil {
		return "", fmt.Errorf("failed to add content to IPFS: %v", err)
	}

	// Pin content with timeout
	pinCtx, pinCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pinCancel()

	err = m.coreAPI.Pin().Add(pinCtx, cidPath)
	if err != nil {
		return "", fmt.Errorf("failed to pin content: %v", err)
	}

	// Get CID string
	cid := cidPath.String()

	return cid, nil
}

// GetFile retrieves a file from IPFS by CID and saves it to outputPath
func (m *IpfsServer) GetFile(cidStr string, outputPath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return fmt.Errorf("IPFS node is not running")
	}

	// Ensure we have a clean CID string
	if strings.HasPrefix(cidStr, "/ipfs/") {
		cidStr = strings.TrimPrefix(cidStr, "/ipfs/")
	}

	// Create necessary directories for the output file
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to resolve the path
	p, err := path.NewPath("/ipfs/" + cidStr)
	if err != nil {
		return fmt.Errorf("failed to parse CID: %v", err)
	}

	// Try to get the file using CoreAPI
	node, err := m.coreAPI.Unixfs().Get(ctx, p)
	if err == nil {
		file, ok := node.(files.File)
		if ok {
			// Create output file
			outFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("failed to create output file: %v", err)
			}
			defer outFile.Close()

			// Copy data from IPFS file to output file
			_, err = io.Copy(outFile, file)
			if err == nil {
				return nil // Success!
			}
		}
	}

	// If CoreAPI fails, try CLI as fallback
	ipfsBin, err := exec.LookPath("ipfs")
	if err == nil {
		// Create a command with timeout
		cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cmdCancel()

		cmd := exec.CommandContext(cmdCtx, ipfsBin, "cat", cidStr)
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outFile.Close()

		cmd.Stdout = outFile
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err = cmd.Run()
		if err == nil {
			return nil // Success with CLI
		}
		log.Printf("CLI cat failed: %v - %s", err, stderr.String())
	}

	return fmt.Errorf("failed to retrieve file for CID %s - all methods failed", cidStr)
}

func (m *IpfsServer) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return nil
	}

	err := m.ipfs.Close()
	if err != nil {
		return fmt.Errorf("failed to stop IPFS node: %v", err)
	}

	m.running = false
	return nil
}

// Unpin method needs to avoid calling IsRunning() while holding the lock
func (m *IpfsServer) Unpin(cidStr string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if IPFS is running directly without calling IsRunning()
	if !m.running {
		return fmt.Errorf("IPFS server is not running")
	}

	// Clean the CID string
	cidStr = strings.TrimSpace(cidStr)

	// Try using CoreAPI first with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse CID
	p, err := path.NewPath("/ipfs/" + cidStr)
	if err != nil {
		return fmt.Errorf("failed to parse CID: %v", err)
	}

	// Try to unpin using CoreAPI
	err = m.coreAPI.Pin().Rm(ctx, p)
	if err == nil {
		return nil
	}

	// Fall back to CLI if CoreAPI fails
	ipfsBin, err := exec.LookPath("ipfs")
	if err == nil {
		cmd := exec.Command(ipfsBin, "pin", "rm", cidStr)
		cmd.Stdout = io.Discard
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err == nil {
			return nil
		}
		log.Printf("Warning: IPFS CLI pin rm failed: %v - %s", err, stderr.String())
	}

	// Not a fatal error - continue
	return nil
}

// Pin pins content to IPFS by its CID
// This method is used by the enclosures module to ensure content is pinned
func (m *IpfsServer) Pin(cidStr string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return fmt.Errorf("IPFS node is not running")
	}

	// Ensure we have a clean CID string
	if strings.HasPrefix(cidStr, "/ipfs/") {
		cidStr = strings.TrimPrefix(cidStr, "/ipfs/")
	}

	// Try using CLI - it's the most reliable method
	ipfsBin, err := exec.LookPath("ipfs")
	if err == nil {
		// IPFS CLI found, use it
		cmd := exec.Command(ipfsBin, "pin", "add", cidStr)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err == nil {
			// Successfully pinned using CLI
			return nil
		}
		// CLI failed, log warning
		fmt.Printf("Warning: CLI pin failed: %v\n", err)
	}

	// If CLI is not available or failed, we'll consider it a non-fatal error
	// since many operations will still work without pinning
	return nil
}

// NewIpfsServer creates a new IPFS server with the default repo path
func NewIpfsServer(repoPath string) (*IpfsServer, error) {
	if repoPath == "" {
		repoPath = filepath.Join("./ipfs-repo")
	}

	m := &IpfsServer{
		repoPath: repoPath,
	}

	return m, nil
}

// GetPeers returns a list of connected peers
func (m *IpfsServer) GetPeers(ctx context.Context) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.running {
		return nil, fmt.Errorf("IPFS node is not running")
	}

	// Create a map to store unique peers
	peerMap := make(map[string]bool)

	// Get peers from PeerHost.Network().Peers()
	for _, p := range m.ipfs.PeerHost.Network().Peers() {
		peerMap[p.String()] = true
	}

	// Get connected peers
	for _, conn := range m.ipfs.PeerHost.Network().Conns() {
		remotePeer := conn.RemotePeer().String()
		peerMap[remotePeer] = true
	}

	// Convert map to slice
	connectedPeers := make([]string, 0, len(peerMap))
	for peer := range peerMap {
		connectedPeers = append(connectedPeers, peer)
	}

	return connectedPeers, nil
}

/*
// ListPinnedItems returns a map of file paths to their CIDs for pinned items
func (is *IpfsServer) ListPinnedItems() (map[string]string, error) {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	// Check if IPFS is running
	if !is.IsRunning() {
		return nil, fmt.Errorf("IPFS server is not running")
	}

	// Initialize an empty map
	items := make(map[string]string)

	// Get list of files we've tracked in our local database
	// This maps file paths to CIDs
	for path, cid := range is.fileCids {
		items[path] = cid
	}

	return items, nil
}

// GetEnclosureInfo attempts to retrieve enclosure information from a CID
func (is *IpfsServer) GetEnclosureInfo(cid string) (*enclosures.Enclosure, error) {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	// Check if IPFS is running
	if !is.IsRunning() {
		return nil, fmt.Errorf("IPFS server is not running")
	}

	// Create a temp file to store the enclosure JSON
	tmpFile, err := os.CreateTemp("", "enclosure-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Try to get the file from IPFS
	err = is.GetFile(cid, tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get enclosure data: %v", err)
	}

	// Read the file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read enclosure data: %v", err)
	}

	// Try to parse as enclosure
	enc, err := enclosures.FromJson(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse enclosure data: %v", err)
	}

	return enc, nil
}
*/
