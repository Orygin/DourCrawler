package main

import (
	"encoding/json"
	"flag"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Artist struct {
	Name    string
	Tags    []string
	Years   []string
	Similar []string
}

var year string
var tag string

var artists map[string]bool
var crawled []*Artist
var newartist chan string
var done chan bool
var waitPool chan bool
var count int
var registerArtist chan *Artist

func init() {
	flag.StringVar(&year, "year", "", "Filter by year")
	flag.StringVar(&tag, "tag", "", "Filter by tag")
}
func main() {
	flag.Parse()
	argsWithoutProg := os.Args[1:]

	if argsWithoutProg[0] == "fetch" {
		crawl()
	} else {
		log.Println("Year: " + year + ", Tag: " + tag)
		search(tag, year)
	}
}
func search(tag string, year string) {
	file, err := ioutil.ReadFile("artistsTagged")
	if err != nil {
		log.Println(err)
		return
	}

	var artists []Artist
	json.Unmarshal(file, &artists)
	for _, artist := range artists {
		foundYear, foundTag := false, false

		if year != "" {
			for _, Artyear := range artist.Years {
				if Artyear == year {
					foundYear = true
				}
			}
		} else {
			foundYear = true
		}
		if tag != "" {
			for _, ArtTag := range artist.Tags {
				if ArtTag == tag {
					foundTag = true
				}
			}
		} else {
			foundTag = true
		}
		if foundYear && foundTag {
			log.Println(artist.Name)
		}
	}
}
func crawl() {

	doc, err := goquery.NewDocument("http://www.dourfestival.eu/en/program/lineup/overview/")
	if err != nil {
		log.Println(err)
		return
	}
	artists = make(map[string]bool)
	doc.Find("#galerie a").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		artists[url] = true
	})

	newartist = make(chan string)
	done = make(chan bool)
	registerArtist = make(chan *Artist)
	waitPool = make(chan bool, 10)
	defer close(newartist)
	defer close(done)
	defer close(registerArtist)

	for artist, _ := range artists {
		count++
		go fetch(artist)
	}
OuterLoop:
	for {
		select {
		case newart := <-newartist:
			if artists[newart] != true {
				artists[newart] = true
				count++
				log.Println("new:" + newart)
				go fetch(newart)
			}
		case <-done:
			count--
			if count == 0 {
				log.Println(count)
				break OuterLoop
			}
		case art := <-registerArtist:

			crawled = append(crawled, art)
		}
	}
	log.Println("ending")
	j, err := json.Marshal(crawled)
	if err != nil {
		return
	}
	file, err := os.Create("artistsTagged")
	if err != nil {
		return
	}
	file.Write(j)
	file.Close()
}

func fetch(artist string) {
	waitPool <- true
	log.Println("fetching:" + artist)
	doc, err := goquery.NewDocument("http://www.dourfestival.eu" + artist)
	if err != nil {
		log.Println(err)
		<-waitPool
		return
	}
	arti := new(Artist)
	arti.Name = artist
	doc.Find("#artiste h1").Each(func(i int, s *goquery.Selection) {
		arti.Name = s.Text()
	})
	doc.Find(".tags li").Each(func(i int, s *goquery.Selection) {
		tag := strings.Replace(s.Text(), "\t", "", -1)
		arti.Tags = append(arti.Tags, tag)
	})
	doc.Find(".yearshow a").Each(func(i int, s *goquery.Selection) {
		arti.Years = append(arti.Years, s.Text())
	})
	doc.Find(".similar a").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		arti.Similar = append(arti.Similar, s.Text())
		newartist <- url
	})

	registerArtist <- arti
	<-waitPool
	done <- true
}
