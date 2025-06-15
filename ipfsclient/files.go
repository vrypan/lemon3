package ipfsclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func PinCID(cid string) error {
	resp, err := http.Post(kuboAPI+"/pin/add?arg="+cid, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pin failed: %s", string(rb))
	}
	return nil
}

func CatCID(cid string) ([]byte, error) {
	resp, err := http.Post(kuboAPI+"/cat?arg="+url.QueryEscape(cid), "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cat failed: %s", string(rb))
	}
	return io.ReadAll(resp.Body)
}

func ProvideCIDRecursive(cid string) error {
	reqURL := fmt.Sprintf("%s/routing/provide?arg=%s&recursive=true", kuboAPI, cid)

	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dht/provide failed: %s", string(body))
	}
	return nil
}
