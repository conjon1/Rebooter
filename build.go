package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Post struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Date    string `json:"date"`
	Excerpt string `json:"excerpt"`
	URL     string `json:"url"`
	Author  string `json:"author"`
}

const (
	postsDir   = "public_html/posts"
	jsonOutput = "public_html/posts.json"
	htmlOutput = "public_html/index.html"
	dateFormat = "January 2, 2006"
)

func main() {
	log.Println(" Starting blog build process...")

	posts, err := findAndParsePosts(postsDir)
	if err != nil {
		log.Fatalf("Error parsing posts: %v", err)
	}

	sortPostsByDate(posts)

	if err := writePostsJSON(posts, jsonOutput); err != nil {
		log.Printf("Warning: Could not write JSON: %v", err)
	}

	f, err := os.Create(htmlOutput)
	if err != nil {
		log.Fatalf("Error creating index.html: %v", err)
	}
	defer f.Close()

	// Render the Index component with the posts data
	err = Index(posts).Render(context.Background(), f)
	if err != nil {
		log.Fatalf(" Error rendering template: %v", err)
	}

	log.Printf("\n Nooice! Built site with %d posts.", len(posts))
}


func findAndParsePosts(dir string) ([]Post, error) {
	var posts []Post
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			post, err := parsePostHTML(filepath.Join(dir, entry.Name()))
			if err != nil {
				log.Printf("Skipping %s: %v", entry.Name(), err)
				continue
			}
			posts = append(posts, post)
		}
	}
	return posts, nil
}

func sortPostsByDate(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		t1, _ := time.Parse(dateFormat, posts[i].Date)
		t2, _ := time.Parse(dateFormat, posts[j].Date)
		return t1.After(t2)
	})
}

func writePostsJSON(posts []Post, path string) error {
	data, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func parsePostHTML(path string) (Post, error) {
	f, err := os.Open(path)
	if err != nil {
		return Post{}, err
	}
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		return Post{}, err
	}

	var post Post
	post.ID = strings.TrimSuffix(filepath.Base(path), ".html")
	post.URL = fmt.Sprintf("posts/%s", filepath.Base(path))

	var crawler func(*html.Node)
	crawler = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && n.FirstChild != nil {
				post.Title = n.FirstChild.Data
			} else if n.Data == "meta" {
				name, content := "", ""
				for _, a := range n.Attr {
					if a.Key == "name" { name = a.Val }
					if a.Key == "content" { content = a.Val }
				}
				if name == "date" { post.Date = content }
				if name == "excerpt" { post.Excerpt = content }
				if name == "author" { post.Author = content }
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawler(c)
		}
	}
	crawler(doc)

	if post.Title == "" || post.Date == "" {
		return Post{}, fmt.Errorf("missing title or date metadata")
	}

	// Default author if not specified
	if post.Author == "" {
		post.Author = "Connal McInnis"
	}

	return post, nil
}
