package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	vlc "github.com/adrg/libvlc-go"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db     *sqlx.DB
	schema = `
		CREATE TABLE radio (
			id INTEGER PRIMARY KEY,
    		name text,
    		url text
		)`
	player *vlc.Player
)

type Radio struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	URL  string `db:"url"`
}

func init() {
	reset := flag.Bool("reset", false, "Reset the database, start from scratch")
	if *reset {
	}
}

func main() {
	os.Remove("./vlcradio.db")
	db, err := sqlx.Open("sqlite3", "./vlcradio.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.MustExec(schema)
	tx := db.MustBegin()
	tx.NamedExec("INSERT INTO radio (name, url) VALUES (:name, :url)", &Radio{0, "Radio1", "http://radio1.nl/stream"})
	tx.NamedExec("INSERT INTO radio (name, url) VALUES (:name, :url)", &Radio{0, "Radio2", "http://radio2.nl/stream"})
	tx.Commit()

	if err := vlc.Init("--no-video", "--quiet"); err != nil {
		// if err := vlc.Init("--no-video", "--quiet", "--alsa-audio-device plughw:CARD=NVidia_1,DEV=7"); err != nil {
		log.Fatal(err)
	}
	defer vlc.Release()

	player, err = vlc.NewPlayer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		player.Stop()
		player.Release()
	}()

	http.HandleFunc("/", index)       // set router
	http.HandleFunc("/add", addradio) // set router

	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))
	err = http.ListenAndServe(":9090", nil) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

func index(w http.ResponseWriter, r *http.Request) {
	radios := []Radio{}
	db.Select(&radios, "SELECT * FROM radio ORDER BY name ASC")

	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, radios)
}

func addradio(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		t, _ := template.ParseFiles("addradio.html")
		t.Execute(w, nil)
	} else {
		name := r.FormValue("name")
		url := r.FormValue("url")

		radio := &Radio{0, name, url}

		tx := db.MustBegin()
		tx.NamedExec("INSERT INTO radio (name, url) VALUES (:name, :url)", radio)
		tx.Commit()

		radios := []Radio{}
		db.Select(&radios, "SELECT * FROM radio ORDER BY name ASC")
		fmt.Println(radios)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func playradio(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/play/"):]

	radio := Radio{}
	err := db.Get(&radio, "SELECT * FROM radio WHERE id=$1", id)
	if err != nil {
		http.Redirect(w, r, "/error", http.StatusNotFound)
	}

	fmt.Println("Playing radio", id)

	player.Stop()

	media, err := player.LoadMediaFromURL(radio.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer media.Release()

	// Play
	err = player.Play()
	if err != nil {
		log.Fatal(err)
	}
}
