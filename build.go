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

type AuthorData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Linkedin string `json:"linkedin"`
	Github   string `json:"github"`
}

const (
	postsDir      = "public_html/posts"
	authorsDir    = "public_html/authors"
	jsonOutput    = "public_html/posts.json"
	authorsOutput = "public_html/authors.json"
	htmlOutput    = "public_html/index.html"
	dateFormat    = "January 2, 2006"
)

func main() {
	log.Println("Starting blog build process...")

	posts, err := findAndParsePosts(postsDir)
	if err != nil {
		log.Fatalf("Error parsing posts: %v", err)
	}
	sortPostsByDate(posts)

	authors, err := findAndParseAuthors(authorsDir)
	if err != nil {
		log.Fatalf("Error parsing authors: %v", err)
	}

	if err := writeJSON(posts, jsonOutput); err != nil {
		log.Printf("Warning: Could not write posts JSON: %v", err)
	}
	if err := writeJSON(authors, authorsOutput); err != nil {
		log.Printf("Warning: Could not write authors JSON: %v", err)
	}

	f, err := os.Create(htmlOutput)
	if err != nil {
		log.Fatalf("Error creating index.html: %v", err)
	}
	defer f.Close()

	err = Index(posts, authors).Render(context.Background(), f)
	if err != nil {
		log.Fatalf("❌ Error rendering template: %v", err)
	}

	log.Printf("\n✅ Nooice! Built site with %d posts and %d authors.", len(posts), len(authors))
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
				log.Printf("Skipping post %s: %v", entry.Name(), err)
				continue
			}
			posts = append(posts, post)
		}
	}
	return posts, nil
}

func findAndParseAuthors(dir string) ([]AuthorData, error) {
	var authors []AuthorData
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return authors, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			author, err := parseAuthorHTML(filepath.Join(dir, entry.Name()))
			if err != nil {
				log.Printf("Skipping author %s: %v", entry.Name(), err)
				continue
			}
			authors = append(authors, author)
		}
	}
	return authors, nil
}

func sortPostsByDate(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		t1, _ := time.Parse(dateFormat, posts[i].Date)
		t2, _ := time.Parse(dateFormat, posts[j].Date)
		return t1.After(t2)
	})
}

func writeJSON(data interface{}, path string) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

func parsePostHTML(path string) (Post, error) {
	doc, err := parseHTMLFile(path)
	if err != nil {
		return Post{}, err
	}

	var post Post
	post.ID = strings.TrimSuffix(filepath.Base(path), ".html")
	post.URL = fmt.Sprintf("posts/%s", filepath.Base(path))

	extractMeta(doc, func(name, content string) {
		switch name {
		case "date":
			post.Date = content
		case "excerpt":
			post.Excerpt = content
		case "author":
			post.Author = content
		}
	})

	var crawler func(*html.Node)
	crawler = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			post.Title = n.FirstChild.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawler(c)
		}
	}
	crawler(doc)

	if post.Title == "" || post.Date == "" {
		return Post{}, fmt.Errorf("missing title or date metadata")
	}
	if post.Author == "" {
		post.Author = "UNKNOWN"
	}

	return post, nil
}

func parseAuthorHTML(path string) (AuthorData, error) {
	doc, err := parseHTMLFile(path)
	if err != nil {
		return AuthorData{}, err
	}

	var author AuthorData
	author.ID = strings.TrimSuffix(filepath.Base(path), ".html")

	extractMeta(doc, func(name, content string) {
		switch name {
		case "name":
			author.Name = content
		case "linkedin":
			author.Linkedin = content
		case "github":
			author.Github = content
		}
	})

	if author.Name == "" {
		return AuthorData{}, fmt.Errorf("missing name metadata")
	}

	return author, nil
}

func parseHTMLFile(path string) (*html.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return html.Parse(f)
}

func extractMeta(n *html.Node, callback func(name, content string)) {
	if n.Type == html.ElementNode && n.Data == "meta" {
		name, content := "", ""
		for _, a := range n.Attr {
			if a.Key == "name" {
				name = a.Val
			}
			if a.Key == "content" {
				content = a.Val
			}
		}
		if name != "" && content != "" {
			callback(name, content)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractMeta(c, callback)
	}
}
