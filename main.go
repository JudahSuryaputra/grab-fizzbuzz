package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	mux := http.NewServeMux()
	fizzBuzzHandler := http.HandlerFunc(FizzBuzzHandler)

	//2. /range-fizzbuzz http endpoint
	mux.Handle("/range-fizzbuzz", Logging(fizzBuzzHandler))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// 5. Graceful shutdowns with SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}
	}()

	<-stop

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}

func FizzBuzzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, errFrom := strconv.Atoi(fromStr)
	to, errTo := strconv.Atoi(toStr)

	if errFrom != nil || errTo != nil || from > to || to-from > 100 {
		http.Error(w, "invalid 'from' or 'to' parameters", http.StatusBadRequest)
		return
	}

	//4. use goroutine to increase performance
	results := make([]string, to-from+1)
	done := make(chan bool, to-from+1)

	for i := from; i <= to; i++ {
		go func(i int) {
			results[i-from] = SingleFizzBuzz(i)
			done <- true
		}(i)
	}

	for i := from; i <= to; i++ {
		<-done
	}

	response := strings.Join(results, " ")
	fmt.Fprint(w, response)
	w.WriteHeader(http.StatusOK)
}

// 1. SingleFizzBuzz function
func SingleFizzBuzz(n int) string {
	if n%3 == 0 && n%5 == 0 {
		return "FizzBuzz"
	} else if n%3 == 0 {
		return "Fizz"
	} else if n%5 == 0 {
		return "Buzz"
	}
	return strconv.Itoa(n)
}

// 3. log request, response and latency to STDOUT
func Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		writer := &responseWriter{ResponseWriter: w, body: &bytes.Buffer{}}

		next.ServeHTTP(writer, r)

		duration := time.Now().Sub(startTime)
		log.Printf("path: %v;\n requestBody: %v;\n\n responseStatus: %v;\n responseBody: %v;\n latency: %v.", r.RequestURI, r.Body, writer.statusCode, writer.body, duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rr *responseWriter) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
}

func (rr *responseWriter) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}
