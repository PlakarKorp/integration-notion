package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type NotionExporter struct {
	token  string
	rootID string //TODO : change this to a user friendly name (e.g. "My Notion Page" instead of "1234567890abcdef")
	sync.Mutex
	mapMutex sync.RWMutex     // Added RWMutex for protecting pageIDMap
	timeout  <-chan time.Time // TODO: change this ugly timeout to a more elegant solution
}

func NewNotionExporter(ctx context.Context, options *exporter.Options, name string, config map[string]string) (exporter.Exporter, error) {
	token, ok := config["token"]
	if !ok {
		return nil, fmt.Errorf("missing token in config")
	}
	rootID, ok := config["rootID"]
	if !ok {
		return nil, fmt.Errorf("missing rootID in config")
	}

	return &NotionExporter{
		token:  token,
		rootID: rootID, //rootID must be an existing page ID, this is the page where the files will be exported
	}, nil
}

func (p *NotionExporter) Root() string {
	return p.rootID
}

func (p *NotionExporter) CreateDirectory(pathname string) error {
	// Notion does not support creating directories
	return nil
}

func (p *NotionExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

var pageIDMap = map[string]string{}

func (p *NotionExporter) Close() error {
	pageIDMap = map[string]string{}
	return nil
}

func DebugResponse(resp *http.Response) {
	// debug
	log.Printf("failed to upload file: %d", resp.StatusCode)
	// Read the response body for more details
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, b, "", "\t")
	if err != nil {
		return
	}
	log.Printf("Error response: %s\n", prettyJSON.String())
	// end
}

func (p *NotionExporter) makeRequest(method, url string, payload []byte) (map[string]any, error) {
	req, err := http.NewRequest(method, url, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", NotionVersionHeader)

	p.Lock()
	resp, err := http.DefaultClient.Do(req)
	p.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		DebugResponse(resp)
		return nil, fmt.Errorf("request failed: status code %d", resp.StatusCode)
	}

	p.RestartTimeout()

	jsonData := map[string]any{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.RestartTimeout()
	return jsonData, nil
}

func (p *NotionExporter) AddBlock(payload []byte, pageID string) (string, error) {
	url := fmt.Sprintf("%s/blocks/%s/children", NotionURL, pageID)
	jsonData, err := p.makeRequest("PATCH", url, payload)
	if err != nil {
		return "", err
	}
	blockID := jsonData["results"].([]any)[0].(map[string]any)["id"].(string)
	return blockID, nil
}

func (p *NotionExporter) CreatePage(payload []byte) (string, error) {
	url := fmt.Sprintf("%s/pages", NotionURL)
	jsonData, err := p.makeRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	return jsonData["id"].(string), nil
}

func (p *NotionExporter) UpdatePage(payload []byte, pageID string) error {
	url := fmt.Sprintf("%s/pages/%s", NotionURL, pageID)
	_, err := p.makeRequest("PATCH", url, payload)
	if err != nil {
		return fmt.Errorf("failed to patch page: %w", err)
	}
	return nil
}

func (p *NotionExporter) CreateDatabase(payload []byte) (string, error) {
	url := fmt.Sprintf("%s/databases", NotionURL)
	jsonData, err := p.makeRequest("POST", url, payload)
	if err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}
	return jsonData["id"].(string), nil
}

func (p *NotionExporter) UpdateDatabase(payload []byte, databaseID string) error {
	url := fmt.Sprintf("%s/databases/%s", NotionURL, databaseID)
	_, err := p.makeRequest("PATCH", url, payload)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	return nil
}

func (p *NotionExporter) AddAllBlocks(jsonData []map[string]any, newID, parentType string) error {
	for _, block := range jsonData { //PATCH each block to the page
		if block["type"] == "child_page" {
			if parentType == "block_id" {
				// Special case for child page inside a block
				payload := map[string]any{
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": p.rootID, //TODO: change this to the closest parent 'page' ID (not block ID)
					},
					"properties": map[string]any{},
					"children":   []any{},
				}

				data, err := json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				newPageID, err := p.CreatePage(data)
				if err != nil {
					return fmt.Errorf("failed to post page: %w", err)
				}
				p.mapMutex.Lock()
				pageIDMap[block["id"].(string)] = newPageID
				p.mapMutex.Unlock()

				// Add the link to the new page
				linkBlock := map[string]any{
					"object": "block",
					"type":   "link_to_page",
					"link_to_page": map[string]any{
						"page_id": newPageID, // Use the ID of the newly created page
					},
				}

				payload = map[string]any{
					"children": []any{
						linkBlock,
					},
				}

				data, err = json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("failed to marshal JSON for link block: %w", err)
				}

				_, err = p.AddBlock(data, newID) // Add the link block to the parent
				if err != nil {
					return fmt.Errorf("failed to add link block: %w", err)
				}
			} else {
				payload := map[string]any{}
				payload["parent"] = map[string]any{
					"type":     parentType,
					parentType: newID,
				}
				payload["properties"] = map[string]any{}
				payload["children"] = []any{}
				data, err := json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				newPageID, err := p.CreatePage(data)
				if err != nil {
					return fmt.Errorf("failed to post page: %w", err)
				}
				p.mapMutex.Lock()
				pageIDMap[block["id"].(string)] = newPageID
				p.mapMutex.Unlock()
			}
		} else if block["type"] == "database" { //?? is this correct?
			payload := map[string]any{
				"parent": map[string]any{
					"type":     parentType,
					parentType: newID,
				},
				"properties": block["properties"],
				"title":      block["title"],
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			databaseID, err := p.CreateDatabase(data)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			p.mapMutex.Lock()
			pageIDMap[block["id"].(string)] = databaseID
			p.mapMutex.Unlock()

		} else { //standard block
			payload := map[string]any{
				"children": []any{
					block,
				},
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			newBlockID, err := p.AddBlock(data, newID)
			if err != nil {
				return fmt.Errorf("failed to patch block: %w", err)
			}
			p.mapMutex.Lock()
			pageIDMap[block["id"].(string)] = newBlockID
			p.mapMutex.Unlock()
		}
	}
	return nil
}

func (p *NotionExporter) RestartTimeout() {
	p.timeout = time.After(5 * time.Second)
}

// StoreFile is outdated and will be removed in the future
func (p *NotionExporter) StoreFile(pathname string, fp io.Reader, size int64) error {

	filetype := filepath.Base(pathname)
	OldID := path.Base(path.Dir(pathname))
	p.RestartTimeout()

	if filetype == "content.json" { //POST empty pages or database to the root page

		var jsonData []map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		// Create a new page for each entry in the JSON array
		for _, entry := range jsonData {
			payload := map[string]any{
				"parent": map[string]any{
					"type":    "page_id",
					"page_id": p.rootID,
				},
				"properties": map[string]any{},
			}
			if entry["object"] == "page" {
				payload["children"] = []any{}
			} else if entry["object"] == "database" {
				payload["properties"] = map[string]any{
					"Name": map[string]any{
						"title": map[string]any{},
					},
				}
			} else {
				return fmt.Errorf("unsupported object type: %s", entry["object"])
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}

			var newId string
			if entry["object"] == "page" {
				newId, err = p.CreatePage(data)
				if err != nil {
					return fmt.Errorf("failed to post page: %w", err)
				}
			} else if entry["object"] == "database" {
				newId, err = p.CreateDatabase(data)
				if err != nil {
					return fmt.Errorf("failed to post database: %w", err)
				}
			}

			p.mapMutex.Lock()
			pageIDMap[entry["id"].(string)] = newId
			p.mapMutex.Unlock()
		}

	} else if filetype == "page.json" { //PATCH header to the page OldID
		newID := func() string {
			for {
				select {
				case <-p.timeout:
					return "" // Timeout reached
				default:
					p.mapMutex.RLock()
					id, ok := pageIDMap[OldID]
					p.mapMutex.RUnlock()
					if ok {
						return id
					}
				}
			}
		}()
		if newID == "" {
			return fmt.Errorf("failed to find new ID for page %s", OldID)
		}

		var jsonData map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		delete(jsonData["properties"].(map[string]any), "Name") //TODO: remove this tmp fix
		jsonData2 := map[string]any{
			"properties": jsonData["properties"],
			"icon":       jsonData["icon"],
			"cover":      jsonData["cover"],
		}
		payload, err := json.Marshal(jsonData2)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		err = p.UpdatePage(payload, newID)
		if err != nil {
			return fmt.Errorf("failed to patch page: %w", err)
		}

		children, ok := jsonData["children"].([]any)
		if !ok || len(children) == 0 {
			log.Printf("%s: children is empty or invalid", OldID)
			return nil
		}

		blocks := make([]map[string]any, len(children))
		for i, child := range children {
			blocks[i] = child.(map[string]any)
		}

		err = p.AddAllBlocks(blocks, newID, "page_id")
		if err != nil {
			return fmt.Errorf("failed to add blocks: %w", err)
		}

	} else if filetype == "blocks.json" { //PATCH blocks to the page OldID

		newID := func() string {
			for {
				select {
				case <-p.timeout:
					return "" // Timeout reached
				default:
					p.mapMutex.RLock()
					id, ok := pageIDMap[OldID]
					p.mapMutex.RUnlock()
					if ok {
						return id
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

		var jsonData []map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		err = p.AddAllBlocks(jsonData, newID, "block_id")
		if err != nil {
			return fmt.Errorf("failed to add blocks: %w", err)
		}

	} else if filetype == "database.json" {

		newDatabaseID := func() string {
			for {
				select {
				case <-p.timeout:
					return "" // Timeout reached
				default:
					p.mapMutex.RLock()
					id, ok := pageIDMap[OldID]
					p.mapMutex.RUnlock()
					if ok {
						return id
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
		if newDatabaseID == "" {
			return fmt.Errorf("failed to find new ID for database %s", OldID)
		}

		var jsonData map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		jsonData2 := map[string]any{
			"properties": jsonData["properties"],
			"icon":       jsonData["icon"],
			"cover":      jsonData["cover"],
		}
		payload, err := json.Marshal(jsonData2)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// Update the existing database
		err = p.UpdateDatabase(payload, newDatabaseID)
		if err != nil {
			return fmt.Errorf("failed to update database: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported file type: %s", filetype)
	}

	return nil
}
