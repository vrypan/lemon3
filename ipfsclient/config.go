package ipfsclient

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// https://docs.ipfs.tech/reference/kubo/rpc/#getting-started
var kuboAPI string

func Init(apiUrl string) {
	kuboAPI = apiUrl
	if err := testConnection(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func testConnection() error {
	url := fmt.Sprintf("%s/id", kuboAPI)
	req, err := http.NewRequest("POST", url, nil)
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
		return fmt.Errorf("%s", string(body))
	}
	return nil
}
