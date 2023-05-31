package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-co-op/gocron"
	"github.com/jackc/pgx"
	"github.com/spf13/viper"
)

type Config struct {
	Urls    []string
	Timeout string
	Db_pass string
	Db_user string
	Db_db   string
	Db_host string
}

var config Config
var conn *pgx.Conn
func ReadConfig() error {

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	err = viper.Unmarshal(&config)
	return err

}

func CheckAddresses() {
	for _, url := range config.Urls {
		resp, err := http.Get(url)
		connect_failed := false
		status_code := 0

		if err != nil {
			connect_failed = true
			fmt.Println("Unable to perform GET request",err.Error())
		}else {
			status_code = resp.StatusCode
			
		}
		
		if status_code != 200 {
			connect_failed = true
			fmt.Println("URL", url, "Not accessible, code=", status_code)
		}else {
			fmt.Println(url,"Accessible")
		}
		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
		sql, args, err := psql.
			Insert("connect_checks").Columns("url", "connect_failed", "status_code", "time_accessed").
			Values(url, connect_failed, status_code, time.Now()).
			ToSql()

		rows, err := conn.Query(sql, args...)
		if err != nil {
			fmt.Println("Unable to exec DB query", err.Error())
		}
		rows.Close()
	}
}

func main() {
	err := ReadConfig()
	if err != nil {
		fmt.Println("Unable to read config", err.Error())
		os.Exit(1)
	}
	db_url := "postgres://" + config.Db_user + ":" + config.Db_pass + "@" + config.Db_host + ":5432/" + config.Db_db
	fmt.Println("DB URL:",db_url)
	conn, err = pgx.Connect(pgx.ConnConfig{Host: config.Db_host,
		 Port: 5432,
		 User: config.Db_user,
		Password: config.Db_pass,
		Database: config.Db_db,})
	if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
	s := gocron.NewScheduler(time.Local)
	_, err = s.Every(config.Timeout).Do(CheckAddresses)
	if err != nil {
		fmt.Println("Unable to create")
		os.Exit(1)
	}

	
	defer conn.Close()
	

	s.StartBlocking()
}
