package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	catalogMapPath = "db/catalog.map.yml"
	catalogPath    = "db/catalog.json"
	catalogVersion = 1
)

type catalogTag struct {
	Tag  string
	Name string
	Icon string
}

func (t *catalogTag) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&t.Tag)
	}
	var raw struct {
		Tag  string `yaml:"tag"`
		Name string `yaml:"name"`
		Icon string `yaml:"icon"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	t.Tag, t.Name, t.Icon = raw.Tag, raw.Name, raw.Icon
	return nil
}

type catalogCategory struct {
	ID        string       `yaml:"id" json:"id"`
	Name      string       `yaml:"name" json:"name"`
	Sensitive bool         `yaml:"sensitive" json:"sensitive,omitempty"`
	Tags      []catalogTag `yaml:"tags" json:"-"`
}

type catalogService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cat  string `json:"cat"`
	Icon string `json:"icon,omitempty"`
	Src  string `json:"src"`
}

type catalogFile struct {
	Version    int               `json:"version"`
	Generated  string            `json:"generated"`
	Base       string            `json:"base"`
	Categories []catalogCategory `json:"categories"`
	Services   []catalogService  `json:"services"`
}

func buildCatalog(index map[string]string, now time.Time) (int, int) {
	mapData, err := os.ReadFile(catalogMapPath)
	if err != nil {
		fmt.Println(" ", err)
		return 0, 0
	}
	var categories []catalogCategory
	if err := yaml.Unmarshal(mapData, &categories); err != nil {
		fmt.Println(" ", err)
		return 0, 0
	}

	out := catalogFile{Version: catalogVersion, Generated: now.Format(time.RFC3339), Categories: categories}
	mapped := make(map[string]struct{}, len(index))
	var lost []string

	for _, cat := range categories {
		for _, t := range cat.Tags {
			url, ok := index[t.Tag]
			if !ok {
				lost = append(lost, cat.ID+"/"+t.Tag)
				continue
			}
			cut := strings.Index(url, "/source")
			if cut < 0 {
				lost = append(lost, cat.ID+"/"+t.Tag)
				continue
			}
			base := url[:cut+1]
			if out.Base == "" {
				out.Base = base
			} else if out.Base != base {
				fmt.Printf("  расхождение base: %s против %s\n", out.Base, base)
			}
			mapped[t.Tag] = struct{}{}
			svc := catalogService{ID: t.Tag, Name: t.Name, Cat: cat.ID, Src: url[cut+1:]}
			if svc.Name == "" {
				svc.Name = t.Tag
			}
			if t.Icon != "" && t.Icon != t.Tag {
				svc.Icon = t.Icon
			}
			out.Services = append(out.Services, svc)
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Println(" ", err)
		return 0, 0
	}
	os.WriteFile(catalogPath, data, 0644)
	var gzBuf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	gz.Write(data)
	gz.Close()
	os.WriteFile(catalogPath+".gz", gzBuf.Bytes(), 0644)

	unmapped := make([]string, 0, len(index)-len(mapped))
	for tag := range index {
		if _, ok := mapped[tag]; !ok {
			unmapped = append(unmapped, tag)
		}
	}
	sort.Strings(unmapped)
	sort.Strings(lost)
	fmt.Printf("  %d категорий, %d сервисов, вне каталога %d тегов, потеряно %d\n",
		len(out.Categories), len(out.Services), len(unmapped), len(lost))
	writeCatalogSummary(unmapped, lost)

	return len(out.Categories), len(out.Services)
}

func writeCatalogSummary(unmapped, lost []string) {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "## Каталог\n\n")
	if len(lost) > 0 {
		fmt.Fprintf(f, "### Теги карты, которых нет в database.json (%d)\n\n%s\n\n", len(lost), strings.Join(lost, ", "))
	}
	fmt.Fprintf(f, "### Теги вне каталога (%d)\n\n%s\n", len(unmapped), strings.Join(unmapped, ", "))
}

func runCatalogOnly() {
	data, err := os.ReadFile("db/database.json")
	if err != nil {
		fmt.Println(" ", err)
		os.Exit(1)
	}
	var index map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		fmt.Println(" ", err)
		os.Exit(1)
	}
	fmt.Println("=== Генерация db/catalog.json ===")
	buildCatalog(index, time.Now().UTC())
}
