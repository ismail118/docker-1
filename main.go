package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"os"
	"time"
)

type Categories struct {
	Id   int64
	Name string
}

func NewConnectionDb() *sql.DB {
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	db, err := sql.Open("mysql", fmt.Sprintf("root:root@tcp(%s:%s)/my_golang_app", host, port))
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err2 := db.Ping(); err2 != nil {
		panic(err2)
	}
	return db
}

func FindAllCategoriesRepo(ctx context.Context, db *sql.DB) []Categories {
	sqlQuery := "SELECT id, name FROM categories"

	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []Categories
	for rows.Next() {
		var categories Categories
		err = rows.Scan(&categories.Id, &categories.Name)
		if err != nil {
			panic(err)
		}
		result = append(result, categories)
	}

	return result
}

func NewRedisClient() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}
	return rdb
}

func main() {
	mux := http.NewServeMux()
	db := NewConnectionDb()
	rdb := NewRedisClient()

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		appName := os.Getenv("APP_NAME")
		fmt.Fprintln(writer, "hello world", appName)
	})

	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		response := FindAllCategoriesRepo(context.Background(), db)
		resByte, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}

		fmt.Fprintln(w, string(resByte))
	})

	mux.HandleFunc("/redis", func(w http.ResponseWriter, r *http.Request) {
		rdb.Set(context.Background(), "key1", "hello from redis", 0)
		result, err := rdb.Get(context.Background(), "key1").Result()
		if err != nil {
			panic(err)
		}

		fmt.Fprintln(w, result)
	})

	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
