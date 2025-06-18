package lemon3libs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/vrypan/lemon3/ipfsclient"
)

type Lemon3Metadata struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Filename    string            `json:"filename"`
	Size        int64             `json:"size"`
	Enclosed    map[string]string `json:"enclosed"`
	Artwork     map[string]string `json:"artwork"`
}

func (m *Lemon3Metadata) ToJSON() []byte {
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return data
}

/*
Given a lemon3 DAG CID, fetch the data from IPFS and return
a Lemon3Metadata object.
*/
func FromCid(cid string) (*Lemon3Metadata, error) {

	if !ipfsclient.Initialized() {
		fmt.Println("Error: lemon3libs.FromCid called without initializing ipfsclient.")
		os.Exit(1)
	}
	metadata, err := ipfsclient.DagGet(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DAG: %w", err)
	}

	// Extract and validate fields
	enclosedField, ok := metadata["enclosed"]
	if !ok {
		return nil, errors.New("DAG does not contain 'enclosed' field")
	}
	enclosedMap, ok := enclosedField.(map[string]any)
	if !ok {
		return nil, errors.New("'enclosed' field is not the expected map structure")
	}
	enclosed, ok := enclosedMap["/"].(string)
	if !ok {
		return nil, errors.New("DAG does not contain valid 'enclosed' CID in 'enclosed' field")
	}

	// Optional: Artwork
	var artwork string
	if artworkField, ok := metadata["artwork"]; ok {
		if artworkMap, ok := artworkField.(map[string]any); ok {
			if val, ok := artworkMap["/"].(string); ok {
				artwork = val
			}
		}
	}

	// Other fields
	title, _ := metadata["title"].(string)
	description, _ := metadata["description"].(string)
	mimeType, _ := metadata["type"].(string)
	filename, _ := metadata["filename"].(string)
	if filename == "" {
		filename = enclosed // fallback
	}
	size, _ := metadata["size"].(int64)
	if size == 0 {
		// JSON numbers come as float64 — handle conversion safely
		if f, ok := metadata["size"].(float64); ok {
			size = int64(f)
		}
	}

	return &Lemon3Metadata{
		Title:       title,
		Description: description,
		Type:        mimeType,
		Filename:    filename,
		Size:        size,
		Enclosed:    map[string]string{"/": enclosed},
		Artwork:     map[string]string{"/": artwork},
	}, nil
}
