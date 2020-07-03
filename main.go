package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type weatherData struct {
	Name string `json:"name"`
	Main struct {
		Kelvin float64 `json:"temp"`
	} `json:"main"`
}

type weatherProvider interface {
	temperature(city string) (float64, error)
}

type openWeatherMap struct {
	url string
}

type weatherBit struct {
	url string
}

type multiWeatherProvider []weatherProvider

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mw := multiWeatherProvider{
		openWeatherMap{
			url: fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?APPID=%s&q=", os.Getenv("OPENWEATHER"))},
		weatherBit{url: fmt.Sprintf("https://api.weatherbit.io/v2.0/current?key=%s&city=", os.Getenv("WEATHERBIT"))},
	}

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		data, err := mw[1].temperature(city)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(data)

	})
	http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello!"))
}

func query(url, city string) (weatherData, error) {
	resp, err := http.Get(city)

	if err != nil {
		return weatherData{}, err
	}

	defer resp.Body.Close()

	var d weatherData

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return weatherData{}, err
	}

	return d, nil
}

func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get(w.url + city)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)

	return d.Main.Kelvin, nil
}

func (w weatherBit) temperature(city string) (float64, error) {
	resp, err := http.Get(w.url + city)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Data []struct {
			Celsius float64 `json:"temp"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	kelvin := d.Data[0].Celsius + 273.15
	log.Printf("weatherBit: %s: %.2f", city, kelvin)
	return kelvin, nil
}

func temperature(city string, providers ...weatherProvider) (float64, error) {
	sum := 0.0
	for _, provider := range providers {
		k, err := provider.temperature(city)
		if err != nil {
			return 0, nil
		}

		sum += k
	}

	return sum / float64(len(providers)), nil
}

func (w multiWeatherProvider) temperature(city string) (float64, error) {
	sum := 0.0

	for _, provider := range w {
		k, err := provider.temperature(city)

		if err != nil {
			return 0, err
		}

		sum += k
	}

	return sum / float64(len(w)), nil
}
