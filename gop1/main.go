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

func downloadPDF(groupID, apiKey, itemKey, fileName string) error {
	url := fmt.Sprintf("https://api.zotero.org/groups/%s/items/%s/file", groupID, itemKey)
	fmt.Println("url used", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Zotero-API-Key", apiKey)
	req.Header.Set("Accept", "application/pdf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed %d", resp.StatusCode)
	}

	folder := "pdfs"
	if err := os.MkdirAll(folder, 0755); err != nil {
		return err
	}

	cleanName := strings.ReplaceAll(fileName, "/", "_")
	cleanName = filepath.Clean(cleanName)
	filePath := filepath.Join(folder, cleanName)

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func processItemsInCollection(groupID, collectionKey, apiKey string, pdfTitles *[]string) {
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

	for _, item := range items {
		if item.Data.ItemType == "attachment" && strings.HasSuffix(strings.ToLower(item.Data.Title), ".pdf") {
			fmt.Printf("  pdf: %s\n", item.Data.Title)
			err := downloadPDF(groupID, apiKey, item.Data.Key, item.Data.Title)
			if err != nil {
				fmt.Println("    error", err)
			} else {
				*pdfTitles = append(*pdfTitles, item.Data.Title)
			}
			continue
		}

		childrenURL := fmt.Sprintf("https://api.zotero.org/groups/%s/items/%s/children", groupID, item.Data.Key)
		var children []Attachment
		if err := fetchJSON(childrenURL, apiKey, &children); err != nil {
			fmt.Println("error", err)
			continue
		}
		for _, att := range children {
			if att.Data.ContentType == "application/pdf" {
				fmt.Printf("  ✅ %s → PDF: %s\n", item.Data.Title, att.Data.Title)
				err := downloadPDF(groupID, apiKey, att.Key, att.Data.Title)
				if err != nil {
					fmt.Println("error:", err)
				} else {
					*pdfTitles = append(*pdfTitles, att.Data.Title)
				}
				break
			}
		}
	}
}

func main() {
	_ = godotenv.Load()
	apiKey := os.Getenv("API_KEY")
	groupID := os.Getenv("GROUP_ID")
	if apiKey == "" || groupID == "" {
		fmt.Println("env is wrong")
		return
	}

	url := fmt.Sprintf("https://api.zotero.org/groups/%s/collections?limit=100", groupID)
	var allCollections []struct {
		Data Collection `json:"data"`
	}
	if err := fetchJSON(url, apiKey, &allCollections); err != nil {
		fmt.Println("error collection", err)
		return
	}

	var datasetKey string
	for _, col := range allCollections {
		if col.Data.Name == "Dataset-Photovoltaics" {
			datasetKey = col.Data.Key
			fmt.Println("parent ", datasetKey)
			break
		}
	}

	if datasetKey == "" {
		fmt.Println("dne")
		return
	}

	level1Subs := getSubcollections(groupID, datasetKey, apiKey)
	pdfTitles := []string{}

	for _, sub1 := range level1Subs {
		fmt.Printf("\nUser: %s\n", sub1.Name)
		processItemsInCollection(groupID, sub1.Key, apiKey, &pdfTitles)

		level2Subs := getSubcollections(groupID, sub1.Key, apiKey)
		for _, sub2 := range level2Subs {
			fmt.Printf("inner %s\n", sub2.Name)
			processItemsInCollection(groupID, sub2.Key, apiKey, &pdfTitles)
		}
	}
}
