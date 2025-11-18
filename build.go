package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Post struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Date    string `json:"date"`
	Excerpt string `json:"excerpt"`
	URL     string `json:"url"`
}

const (
	postsDir   = "public_html/posts"
	outputFile = "public_html/posts.json"
	dateFormat = "January 2, 2006"
)

func main() {
	log.Println("Starting blog post build process...")

	posts, err := findAndParsePosts(postsDir)
	if err != nil {
		log.Fatalf("Error finding or parsing posts: %v", err)
	}

	sortPostsByDate(posts)

	err = writePostsJSON(posts, outputFile)
	if err != nil {
		log.Fatalf("Error writing JSON file: %v", err)
	}

	log.Printf("\n Success! Created %s with %d posts.", outputFile, len(posts))
}

func findAndParsePosts(dir string) ([]Post, error) {
	var posts []Post
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("could not read posts directory '%s': %w", dir, err)
	}

	log.Println("Scanning for posts in", dir)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			filePath := filepath.Join(dir, file.Name())
			post, err := parsePostHTML(filePath)
			if err != nil {
				log.Printf(" you done fuckd up  Skipping %s: could not parse metadata: %v", file.Name(), err)
				continue
			}
			posts = append(posts, post)
			log.Printf("  - Found and parsed: %s", file.Name())
		}
	}
	return posts, nil
}

func sortPostsByDate(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		t1, err := time.Parse(dateFormat, posts[i].Date)
		if err != nil {
			// Fail loudly
			log.Fatalf("FATAL: Could not parse date '%s' for post '%s'. Please use '%s' format. Error: %v", posts[i].Date, posts[i].Title, dateFormat, err)
		}
		t2, err := time.Parse(dateFormat, posts[j].Date)
		if err != nil {
			log.Fatalf("FATAL: Could not parse date '%s' for post '%s'. Please use '%s' format. Error: %v", posts[j].Date, posts[j].Title, dateFormat, err)
		}
		return t1.After(t2)
	})
}

func writePostsJSON(posts []Post, path string) error {
	jsonData, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal posts to JSON: %w", err)
	}
	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("could not write to file '%s': %w", path, err)
	}
	return nil
}

func parsePostHTML(filePath string) (Post, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Post{}, err
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		return Post{}, err
	}

	var post Post
	post.ID = strings.TrimSuffix(filepath.Base(filePath), ".html")
	post.URL = fmt.Sprintf("posts/%s", filepath.Base(filePath))

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					post.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				var name, content string
				for _, a := range n.Attr {
					if a.Key == "name" {
						name = a.Val
					}
					if a.Key == "content" {
						content = a.Val
					}
				}
				switch name {
				case "date":
					post.Date = content
				case "excerpt":
					post.Excerpt = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if post.Title == "" || post.Date == "" || post.Excerpt == "" {
		return Post{}, fmt.Errorf("missing required metadata (title, date, or excerpt)")
	}

	return post, nil
}
