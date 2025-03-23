package enclosure

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/vrypan/lemon3/farcaster"
	"github.com/vrypan/lemon3/fctools"
	db "github.com/vrypan/lemon3/localdb"
)

// IpfsUploader interface defines methods needed for IPFS interaction
type IpfsUploader interface {
	AddFile(filePath string) (string, error)
	AddBytes(content []byte) (string, error)
	GetFile(cidStr string, outputPath string) error
	Pin(cidStr string) error
}

// Enclosure represents a media enclosure with metadata
type Enclosure struct {
	FileType     string `json:"mime_type"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"size"`
	FileCID      string `json:"file_cid"`
	Description  string `json:"description"`
	EnclosureCID string `json:"enclosure_cid,omitempty"`
}

func NewEnclosure(ipfs IpfsUploader, filePath, fileName, fileType, description string) (*Enclosure, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("error accessing file: %w", err)
	}

	// Get file size
	size := fileInfo.Size()

	// Guess the file mime type if not provided
	if fileType == "" {
		fileType, err = getFileMimeType(filePath)
		if err != nil {
			return nil, fmt.Errorf("error guessing file mime type: %w", err)
		}
	}

	// Upload the file to IPFS
	fileCID, err := ipfs.AddFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error uploading file to IPFS: %w", err)
	}
	fileCID = strings.TrimPrefix(fileCID, "/ipfs/")

	// Create enclosure with file metadata
	enclosure := &Enclosure{
		FileType:    fileType,
		FileName:    fileName,
		Description: description,
		FileCID:     fileCID,
		FileSize:    size,
	}

	// Create a temporary JSON file with the enclosure data
	jsonData, err := json.MarshalIndent(enclosure, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating JSON: %w", err)
	}

	// Upload the JSON to IPFS
	enclosureCID, err := ipfs.AddBytes(jsonData)
	if err != nil {
		return nil, fmt.Errorf("error uploading JSON to IPFS: %w", err)
	}

	// Set the enclosure CID
	enclosureCID = strings.TrimPrefix(enclosureCID, "/ipfs/")
	enclosure.EnclosureCID = enclosureCID

	return enclosure, nil
}
func NewEnclosureFromCID(ipfs IpfsUploader, cid string) (*Enclosure, error) {
	// Create temporary file to store the JSON metadata
	tmpDir := os.TempDir()
	jsonFilePath := filepath.Join(tmpDir, fmt.Sprintf("enclosure-%s.json", cid))

	// Fetch the enclosure JSON from IPFS
	err := ipfs.GetFile(cid, jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("error fetching enclosure metadata from IPFS: %w", err)
	}

	// Read the JSON file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		os.Remove(jsonFilePath)
		return nil, fmt.Errorf("error reading enclosure JSON: %w", err)
	}

	// Clean up the temporary file
	os.Remove(jsonFilePath)

	// Parse the JSON into an enclosure
	var enclosure Enclosure
	err = json.Unmarshal(jsonData, &enclosure)
	if err != nil {
		return nil, fmt.Errorf("error parsing enclosure JSON: %w", err)
	}

	// Set the enclosure CID (in case it wasn't included in the JSON)
	enclosure.EnclosureCID = cid

	// Pin the enclosure CID
	err = ipfs.Pin(cid)
	if err != nil {
		return nil, fmt.Errorf("error pinning enclosure CID: %w", err)
	}

	// Pin the file CID
	err = ipfs.Pin(enclosure.FileCID)
	if err != nil {
		return nil, fmt.Errorf("error pinning file CID: %w", err)
	}

	return &enclosure, nil
}

// Json returns the enclosure in JSON format
func (e *Enclosure) Json() ([]byte, error) {
	jsonData, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating JSON: %w", err)
	}

	return jsonData, nil
}

func (e *Enclosure) HumanReadableSize() string {
	size := e.FileSize
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
}

// FromJson creates an Enclosure from a JSON string
func FromJson(jsonStr string) (*Enclosure, error) {
	var enclosure Enclosure

	err := json.Unmarshal([]byte(jsonStr), &enclosure)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &enclosure, nil
}

// FromCID creates an Enclosure by fetching its data from IPFS
func FromCID(ipfs IpfsUploader, cid string) (*Enclosure, error) {
	db.AssertOpen()
	dbkey := []byte(fmt.Sprintf("enclosure:%s", cid))
	data, err := db.Get(dbkey)
	if err != nil {
		enc, err := NewEnclosureFromCID(ipfs, cid)
		if err != nil {
			return nil, fmt.Errorf("%s not found", cid)
		}
		json, err := enc.Json()
		if err != nil {
			return nil, fmt.Errorf("error marshaling JSON: %w", err)
		}
		db.Set(dbkey, json)
		return enc, nil
	}
	enc, err := FromJson(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	return enc, nil
}

func getFileMimeType(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the first 512 bytes to determine the file type
	byteSlice := make([]byte, 512)
	_, err = file.Read(byteSlice)
	if err != nil {
		return "", err
	}

	// Use the net/http package's DetectContentType function
	// to find the MIME/Content Type of the file
	contentType := http.DetectContentType(byteSlice)

	return contentType, nil
}

func (e *Enclosure) Post(hub *fctools.FarcasterHub, fid uint64, privateKey []byte) (*pb.MessageData, error) {
	if hub == nil {
		return nil, errors.New("hub is nil")
	}
	body := fmt.Sprintf("%s\n%s\n\n%s", e.FileName, e.Description, "[with lemon3]")
	link := "enclosure+ipfs://" + e.EnclosureCID
	cast, err := hub.SendCast(fid, privateKey, body, link)
	if err != nil {
		return nil, fmt.Errorf("error uploading file to IPFS: %w", err)
	}
	return cast, nil
}
