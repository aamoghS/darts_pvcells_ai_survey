package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Collection struct {
	Key              string      `json:"key"`
	ParentCollection interface{} `json:"parentCollection"`
	Name             string      `json:"name"`
}

type Item struct {
	Key  string `json:"key"`
	Data struct {
		Title    string `json:"title"`
		ItemType string `json:"itemType"`
	} `json:"data"`
}

type Attachment struct {
	Key  string `json:"key"`
	Data struct {
		ContentType string `json:"contentType"`
		Title       string `json:"title"`
	} `json:"data"`
}

func fetchJSON(url, apiKey string, target interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Zotero-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func getSubcollections(groupID, parentKey, apiKey string) []Collection {
	url := fmt.Sprintf("https://api.zotero.org/groups/%s/collections?limit=1000", groupID)
	var allCollections []struct {
		Data Collection `json:"data"`
	}
	if err := fetchJSON(url, apiKey, &allCollections); err != nil {
		fmt.Println("Error fetching collections:", err)
		return nil
	}

	subs := []Collection{}
	for _, c := range allCollections {
		if str, ok := c.Data.ParentCollection.(string); ok && str == parentKey {
			subs = append(subs, c.Data)
		}
	}
	return subs
}

func downloadPDF(groupID, apiKey, itemKey, filePath string) error {
	url := fmt.Sprintf("https://api.zotero.org/groups/%s/items/%s/file", groupID, itemKey)
	fmt.Println("Downloading:", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Zotero-API-Key", apiKey)
	req.Header.Set("Accept", "application/pdf")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Set("Zotero-API-Key", apiKey)
			req.Header.Set("Accept", "application/pdf")
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func fetchBibLaTeX(groupID, itemKey, apiKey string) (string, error) {
	url := fmt.Sprintf("https://api.zotero.org/groups/%s/items/%s?format=biblatex", groupID, itemKey)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Zotero-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch BibLaTeX (status: %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return filepath.Clean(name)
}

func processItemsInCollection(groupID, collectionKey, apiKey, collectionName string) {
	itemsURL := fmt.Sprintf("https://api.zotero.org/groups/%s/collections/%s/items?limit=100", groupID, collectionKey)
	var items []struct {
		Data struct {
			Key      string `json:"key"`
			Title    string `json:"title"`
			ItemType string `json:"itemType"`
		} `json:"data"`
	}
	if err := fetchJSON(itemsURL, apiKey, &items); err != nil {
		fmt.Println("  Error fetching items:", err)
		return
	}

	bibEntries := ""
	folder := filepath.Join("exports", sanitizeFileName(collectionName))
	os.MkdirAll(folder, 0755)

	for _, item := range items {
		pdfDownloaded := false

		if item.Data.ItemType == "attachment" && strings.HasSuffix(strings.ToLower(item.Data.Title), ".pdf") {
			filePath := filepath.Join(folder, item.Data.Title)
			err := downloadPDF(groupID, apiKey, item.Data.Key, filePath)
			if err == nil {
				pdfDownloaded = true
			}
			continue
		}

		childrenURL := fmt.Sprintf("https://api.zotero.org/groups/%s/items/%s/children", groupID, item.Data.Key)
		var children []Attachment
		if err := fetchJSON(childrenURL, apiKey, &children); err != nil {
			fmt.Println("  error fetching children:", err)
			continue
		}

		for _, att := range children {
			if att.Data.ContentType == "application/pdf" {
				filePath := filepath.Join(folder, att.Data.Title)
				err := downloadPDF(groupID, apiKey, att.Key, filePath)
				if err == nil {
					pdfDownloaded = true
				}
			}
		}

		if pdfDownloaded {
			entry, err := fetchBibLaTeX(groupID, item.Data.Key, apiKey)
			if err == nil {
				bibEntries += entry + "\n\n"
			}
		}
	}

	if bibEntries != "" {
		bibFile := filepath.Join(folder, "collection.bib")
		os.WriteFile(bibFile, []byte(bibEntries), 0644)
	}
}

func main() {
	_ = godotenv.Load()
	apiKey := os.Getenv("API_KEY")
	groupID := os.Getenv("GROUP_ID")
	if apiKey == "" || groupID == "" {
		fmt.Println("Environment variables API_KEY or GROUP_ID are missing")
		return
	}

	url := fmt.Sprintf("https://api.zotero.org/groups/%s/collections?limit=100", groupID)
	var allCollections []struct {
		Data Collection `json:"data"`
	}
	if err := fetchJSON(url, apiKey, &allCollections); err != nil {
		fmt.Println("Error fetching collections:", err)
		return
	}

	var datasetKey string
	for _, col := range allCollections {
		if col.Data.Name == "Dataset-Photovoltaics" {
			datasetKey = col.Data.Key
			fmt.Println("Parent collection found:", datasetKey)
			break
		}
	}

	if datasetKey == "" {
		fmt.Println("Dataset-Photovoltaics collection not found")
		return
	}

	level1Subs := getSubcollections(groupID, datasetKey, apiKey)

	for _, sub1 := range level1Subs {
		fmt.Printf("\nUser: %s\n", sub1.Name)
		processItemsInCollection(groupID, sub1.Key, apiKey, sub1.Name)

		level2Subs := getSubcollections(groupID, sub1.Key, apiKey)
		for _, sub2 := range level2Subs {
			fmt.Printf("  inner: %s\n", sub2.Name)
			processItemsInCollection(groupID, sub2.Key, apiKey, sub2.Name)
		}
	}
}
