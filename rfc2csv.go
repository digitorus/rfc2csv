package main

import (
	"encoding/csv"
	"fmt"
	gohtml "html"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Service implements the RFC to CSV conversion
type Service struct {
	doc    *html.Tokenizer
	save   chan Row
	writer *csv.Writer
	wg     sync.WaitGroup
}

// Row as written to the CSV
type Row struct {
	Category    string
	Name        string
	Title       string
	Description string
	Notes       string
}

var skipSections = []string{"Introduction", "Definitions",
	"IANA Considerations", "Acknowledgements", "References"}

func main() {
	if len(os.Args) < 2 {
		log.Println("usage: " + os.Args[0] + " 5280 ...")
		os.Exit(1)
	}

	for _, rfc := range os.Args[1:] {
		doRFC(rfc)
	}
}

func doRFC(rfc string) {
	url := "https://tools.ietf.org/html/rfc" + rfc
	filename := path.Base(url)

	s := &Service{}
	go s.newWriter(filename)

	err := s.parse(url)
	if err != nil {
		log.Fatal(err)
	}
	s.wg.Wait()
}

func (s *Service) newWriter(filename string) {
	s.wg.Add(1)
	defer s.wg.Done()

	file, err := os.Create(filename + ".csv")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	s.writer = csv.NewWriter(file)
	defer s.writer.Flush()

	err = s.writer.Write([]string{
		"Category", "Name", "Title", "Description", "Notes"})
	if err != nil {
		log.Println(err)
		return
	}

	s.save = make(chan Row)

	for {
		row, more := <-s.save
		if !more {
			// channel closed, stop writer
			return
		}

		err = s.writer.Write([]string{
			row.Category,
			row.Name,
			row.Title,
			row.Description,
			row.Notes,
		})
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}

func (s *Service) parse(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("content type must be html")
	}

	var category string
	s.doc = html.NewTokenizer(resp.Body)
	for tokenType := s.doc.Next(); tokenType != html.ErrorToken; {
		token := s.doc.Token()
		if tokenType == html.StartTagToken {
			if isSection(&token) {
				var row Row
				row.Name = s.getNextText()

				row.Title = s.getNextText()
				row.Title = strings.TrimPrefix(row.Title, ".")
				row.Title = strings.TrimSpace(row.Title)

				if !strings.Contains(row.Name, ".") {
					category = row.Title
				}

				row.Category = category
				row.Description = s.getSection()

				if row.Description != "" && !inSlice(category, skipSections) {
					s.save <- row
				}
			}

		}
		tokenType = s.doc.Next()
	}

	close(s.save)

	err = s.doc.Err()
	if err != io.EOF {
		return err
	}

	return nil
}

func (s *Service) skipToEndToken(t *html.Token) {
	level := 1
	for i := 0; i < 100000; i++ {
		tokenType := s.doc.Next()
		token := s.doc.Token()
		if tokenType == html.StartTagToken && token.DataAtom == t.DataAtom {
			level++
		} else if tokenType == html.EndTagToken && token.DataAtom == t.DataAtom {
			level--
		}
		if level <= 0 {
			break
		}
	}
}

func (s *Service) getNextText() string {
	for i := 0; i < 100000; i++ {
		tokenType := s.doc.Next()
		if tokenType == html.TextToken {
			token := s.doc.Token()
			return gohtml.UnescapeString(token.String())
		}
	}
	return ""
}

func (s *Service) getSection() string {
	var content string
	for i := 0; i < 10000; i++ {
		tokenType := s.doc.Next()
		token := s.doc.Token()
		if isBlocked(&token) {
			if tokenType != html.SelfClosingTagToken {
				s.skipToEndToken(&token)
			}
			continue
		}
		if isSection(&token) {
			break
		}
		if tokenType == html.TextToken {
			content = content + gohtml.UnescapeString(token.String())
		}
	}

	// remove duplicate newlines
	regex, err := regexp.Compile("(?m)^\n\n*")
	if err != nil {
		return content
	}
	content = regex.ReplaceAllString(content, "\n")

	// remove section indentation
	regex, err = regexp.Compile("(?m)^   ")
	if err != nil {
		return content
	}
	content = regex.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

func isSection(t *html.Token) bool {
	if t.DataAtom == atom.A {
		for _, a := range t.Attr {
			if a.Key == "name" && strings.HasPrefix(a.Val, "section") {
				return true
			}
		}
	}
	return false
}

func isBlocked(t *html.Token) bool {
	for _, a := range t.Attr {
		if a.Key == "class" && (a.Val == "invisible" || a.Val == "grey" ||
			strings.Contains(a.Val, "noprint")) {
			return true
		}
	}

	return false
}

func inSlice(key string, slice []string) bool {
	for _, item := range slice {
		if strings.ToLower(key) == strings.ToLower(item) {
			return true
		}
	}
	return false
}
