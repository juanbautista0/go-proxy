package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/scale/up", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("🚀 TRIGGER EJECUTADO: Scale UP - Tráfico alto detectado!")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/scale/down", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("📉 TRIGGER EJECUTADO: Scale DOWN - Tráfico bajo detectado!")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/morning", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("🌅 TRIGGER EJECUTADO: Morning scale - 09:00")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/evening", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("🌙 TRIGGER EJECUTADO: Evening scale - 18:00")
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Mock actions server listening on :9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}