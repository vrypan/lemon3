package ipfsclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// dagPut serializes JSON to DAG-CBOR and stores it
func DagPut(obj map[string]any) (string, error) {
	// Serialize JSON
	payload, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	// Multipart/form-data body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "data.json")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(payload); err != nil {
		return "", err
	}
	writer.Close()

	// POST to /dag/put
	req, err := http.NewRequest("POST", kuboAPI+"/dag/put?store-codec=dag-cbor&input-codec=json", &buf)
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
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("dag/put failed: %s", string(body))
	}

	var result struct {
		Cid struct {
			Root string `json:"/"`
		} `json:"Cid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Cid.Root, nil
}

// dagGet fetches a DAG object as JSON
func DagGet(cid string) (map[string]any, error) {
	url := fmt.Sprintf("%s/dag/get?arg=%s", kuboAPI, cid)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dag/get failed: %s", string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func DagGetPath(path string) (any, error) {
	url := fmt.Sprintf("%s/dag/get?arg=%s", kuboAPI, url.QueryEscape(path))
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dag/get path failed: %s", string(body))
	}

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
