package main

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// #region city
type City struct {
	ID          int    `json:"ID,omitempty" db:"ID"`
	Name        string `json:"name,omitempty" db:"Name"`
	CountryCode string `json:"countryCode,omitempty"  db:"CountryCode"`
	District    string `json:"district,omitempty"  db:"District"`
	Population  int    `json:"population,omitempty"  db:"Population"`
}

type Country struct {
	Code       string `json:"Code,omitempty" db:"Code"`
	Name       string `json:"Name,omitempty" db:"Name"`
	Population int    `json:"Population,omitempty" db:"Population"`
}

// #endregion city
func main() {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatal(err)
	}

	conf := mysql.Config{
		User:      os.Getenv("DB_USERNAME"),
		Passwd:    os.Getenv("DB_PASSWORD"),
		Net:       "tcp",
		Addr:      os.Getenv("DB_HOSTNAME") + ":" + os.Getenv("DB_PORT"),
		DBName:    os.Getenv("DB_DATABASE"),
		ParseTime: true,
		Collation: "utf8mb4_unicode_ci",
		Loc:       jst,
	}

	db, err := sqlx.Open("mysql", conf.FormatDSN())

	if err != nil {
		log.Fatal(err)
	}

	log.Println("connected")
	// #region get
	var city City
	err = db.Get(&city, "SELECT * FROM city WHERE Name = ?", "Tokyo")
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such city Name = '%s'\n", "Tokyo")
		return
	}
	if err != nil {
		log.Fatalf("DB Error: %s\n", err)
	}
	// #endregion get
	log.Printf("Tokyoの人口は%d人です\n", city.Population)

	// 基本問題
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s cityName\n", os.Args[0])
	}
	cityName := os.Args[1]
	err = db.Get(&city, "SELECT * FROM city WHERE Name = ?", cityName)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such city Name = '%s'\n", cityName)
		return
	}
	if err != nil {
		log.Fatalf("DB Error: %s\n", err)
	}
	log.Printf("%sの人口は%d人です\n", cityName, city.Population)
	var country Country
	err = db.Get(&country, "SELECT Code, Name, Population FROM country WHERE Code = ?", city.CountryCode)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such country Code = '%s'\n", city.CountryCode)
		return
	}
	if err != nil {
		log.Fatalf("DB Error: %s\n", err)
	}
	log.Printf("%sの人口は%d人です\n", country.Name, country.Population)
	log.Printf("%sの人口比率は%.2f%%です\n", cityName, float64(city.Population)/float64(country.Population)*100)
}