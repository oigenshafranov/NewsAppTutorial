package main

import (
	"bytes"
	// "fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/oigenshafranov/NewsAppTutorial/news"
)

var tpl = template.Must(template.ParseFiles("index.html"))

type Search struct {
	Query      string
	NextPage   int
	TotalPages int
	Results    *news.Results
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
	// tpl.Execute(w, nil)
}

func searchHandler(newsapi *news.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search := &Search{}
		u, err := url.Parse(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		params := u.Query()
		search.Query = params.Get("q")

		if len(search.Query) == 0 {
			search := &Search{
				Results: &news.Results{},
			}

			buf := &bytes.Buffer{}
			err = tpl.Execute(buf, search)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			buf.WriteTo(w)
			return
		}
		page := params.Get("page")
		if page == "" {
			page = "1"
		}

		results, err := newsapi.FetchEverything(search.Query, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nextPage, err := strconv.Atoi(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		search.NextPage = nextPage
		search.TotalPages = int(math.Ceil(float64(results.TotalResults) / float64(newsapi.PageSize)))
		search.Results = results

		if ok := !search.IsLastPage(); ok {
			search.NextPage++
		}

		buf := &bytes.Buffer{}
		err = tpl.Execute(buf, search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		buf.WriteTo(w)

		// fmt.Printf("%+v", results)

	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	myClient := &http.Client{Timeout: 10 * time.Second}
	newsapi := news.NewClient(myClient, apiKey, 20)

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()

	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/search", searchHandler(newsapi))
	mux.HandleFunc("/", indexHandler)
	http.ListenAndServe(":"+port, mux)
}
