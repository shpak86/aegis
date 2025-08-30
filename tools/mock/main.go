package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"
)

type Comment struct {
	ID      int    `json:"id"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Content string `json:"content"`
	Likes   int    `json:"likes"`
}

type Article struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	Date          string    `json:"date"`
	Content       string    `json:"content"`
	Likes         int       `json:"likes"`
	CommentsCount int       `json:"commentsCount"`
	Comments      []Comment `json:"comments"`
}

type Repository struct {
	Articles []Article `json:"articles"`
}

func (r *Repository) Load(file string) (err error) {
	var content []byte
	content, err = os.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, r)
	return
}

func (r *Repository) Save(file string) (err error) {
	if content, err := json.Marshal(r); err != nil {
		return err
	} else {
		return os.WriteFile(file, content, 0644)
	}
}

func (r *Repository) GetArticles() []Article {
	return r.Articles
}

func (r *Repository) GetArticle(id int) (art *Article, ok bool) {
	for i := range r.Articles {
		if r.Articles[i].ID == id {
			art = &r.Articles[i]
			ok = true
			return
		}
	}
	return
}

func (r *Repository) LikeArticle(articleId int) bool {
	article, exists := r.GetArticle(articleId)
	if !exists {
		return false
	}
	article.Likes++
	return true
}

func (r *Repository) GetComments(articleId int) (comments []Comment, ok bool) {
	article, ok := r.GetArticle(articleId)
	if !ok {
		return nil, false
	}
	return article.Comments, true
}

func (r *Repository) AddComment(articleId int, comment Comment) bool {
	article, ok := r.GetArticle(articleId)
	if !ok {
		return false
	}
	comment.ID = rand.Intn(1000000)
	comment.Likes = 0
	comment.Date = time.Now().Format("2006-01-02 15:04:05")
	article.Comments = append(article.Comments, comment)
	article.CommentsCount = len(article.Comments)
	return true
}

func (r *Repository) LikeComment(articleId, commentId int) bool {
	article, exists := r.GetArticle(articleId)
	if !exists {
		return false
	}
	for i := range article.Comments {
		if commentId == article.Comments[i].ID {
			article.Comments[i].Likes++
			return true
		}
	}
	return false
}

func main() {
	listenPort := flag.Int("listen", 8081, "listen port")
	staticDir := flag.String("static", "static", "static directory")
	model := flag.String("model", "default.json", "model path")
	flag.Parse()

	repository := Repository{}
	err := repository.Load(*model)
	if err != nil {
		slog.Error("load model failed", "error", err)
		os.Exit(1)
	}
	http.HandleFunc("GET /articles/", func(writer http.ResponseWriter, request *http.Request) {
		articles := slices.Clone(repository.GetArticles())
		for i := range articles {
			articles[i].Comments = []Comment{}
		}
		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(articles)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("GET /articles/{id}", func(writer http.ResponseWriter, request *http.Request) {
		var article *Article
		var articleId int
		var ok bool
		id := request.PathValue("id")
		if articleId, err = strconv.Atoi(id); err != nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if article, ok = repository.GetArticle(articleId); !ok {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if err = json.NewEncoder(writer).Encode(article); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.HandleFunc("POST /articles/{id}/like", func(writer http.ResponseWriter, request *http.Request) {
		var articleId int
		var ok bool
		id := request.PathValue("id")
		if articleId, err = strconv.Atoi(id); err != nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if ok = repository.LikeArticle(articleId); !ok {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte("{}"))
	})

	http.HandleFunc("POST /articles/{id}/comments", func(writer http.ResponseWriter, request *http.Request) {
		var articleId int
		var comment Comment
		var ok bool
		id := request.PathValue("id")
		if articleId, err = strconv.Atoi(id); err != nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&comment); err != nil {
			writer.WriteHeader(http.StatusUnprocessableEntity)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if ok = repository.AddComment(articleId, comment); !ok {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte("{}"))
	})

	http.HandleFunc("POST /articles/{aid}/comments/{cid}/like", func(writer http.ResponseWriter, request *http.Request) {
		var articleId int
		var commentId int
		var ok bool
		aid := request.PathValue("aid")
		if articleId, err = strconv.Atoi(aid); err != nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		cid := request.PathValue("cid")
		if commentId, err = strconv.Atoi(cid); err != nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		if ok = repository.LikeComment(articleId, commentId); !ok {
			writer.WriteHeader(http.StatusNotFound)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte("{}"))
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte("{}"))
	})
	staicHandler := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", staicHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", *listenPort), nil)
}
