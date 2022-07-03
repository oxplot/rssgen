package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"gopkg.in/yaml.v3"
)

type FeedSpec struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Link        string `yaml:"link"`
	Spec        struct {
		Item        string `yaml:"item"`
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Link        string `yaml:"link"`
	} `yaml:"spec"`
}

type Config struct {
	Listen string              `yaml:"listen"`
	Feeds  map[string]FeedSpec `yaml:"feeds"`
}

var (
	configPath = flag.String("config", "", "path to config file")
	config     = Config{Listen: "127.0.0.1:9977"}
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf8")
	t := template.Must(template.New("home").Parse(`<!doctype html>
<html>
<h2>Feeds</h2>
<ul>
{{range $slug, $feed := .}}
<li><a href="/feeds/{{$slug}}">{{$feed.Title}}</a></li>
{{end}}
</ul>
</html>
`))
	t.Execute(w, config.Feeds)
}

func handleFeeds(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/feeds/"):]
	feedSpec, ok := config.Feeds[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	res, err := http.Get(feedSpec.Link)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("error fetching feed %s: %s", id, err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("bad status code fetching feed %s: %d", id, res.StatusCode)
		return
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error parsing the feed %s: %s", id, err)
		return
	}

	feed := &feeds.Feed{
		Title:       feedSpec.Title,
		Link:        &feeds.Link{Href: feedSpec.Link},
		Description: feedSpec.Description,
		Items:       []*feeds.Item{},
	}

	base, err := url.Parse(feedSpec.Link)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error parsing the feed %s: %s", id, err)
		return
	}

	doc.Find(feedSpec.Spec.Item).Each(func(i int, s *goquery.Selection) {
		var link string
		desc, _ := s.Find(feedSpec.Spec.Description).Html()
		u, err := base.Parse(s.Find(feedSpec.Spec.Link).AttrOr("href", ""))
		if err != nil {
			link = "failed to parse link: err"
		} else {
			link = u.String()
		}
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       s.Find(feedSpec.Spec.Title).Text(),
			Link:        &feeds.Link{Href: link},
			Description: desc,
		})
	})

	w.Header().Add("Content-Type", "application/rss+xml; charset=utf8")
	feed.WriteRss(w)
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if *configPath != "" {
		var f *os.File
		var err error
		if *configPath == "-" {
			f = os.Stdin
		} else {
			f, err = os.Open(*configPath)
		}
		if err != nil {
			log.Fatalf("failed to open %s: %s", *configPath, err)
		}
		y := yaml.NewDecoder(f)
		if err = y.Decode(&config); err != nil {
			log.Fatalf("failed to load config %s: %s", *configPath, err)
		}
		f.Close()
	}
	log.Printf("listening on http://%s/", config.Listen)
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/feeds/", handleFeeds)
	log.Fatal(http.ListenAndServe(config.Listen, nil))
}
