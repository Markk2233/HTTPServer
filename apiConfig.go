package main

import (
	"awesomeProject/internal/database"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func (config *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		config.fileserverHits.Add(1)
		next.ServeHTTP(writer, req)
	})
}

func (config *apiConfig) getHits(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(200)
	const adminHtml = `<html>
		  <Body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		  </Body>
		</html>`
	fmt.Fprintf(writer, adminHtml, config.fileserverHits.Load())
}

func (config *apiConfig) reset(writer http.ResponseWriter, req *http.Request) {
	config.fileserverHits.Store(0)
}

func (config *apiConfig) checkServer(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("OK\n"))
}

type jsonBody struct {
	Body string `json:"body"`
}

type jsonError struct {
	Err string `json:"error"`
}

func (e jsonError) Error() string {
	return e.Err
}

func (j jsonError) RuntimeError() {
	//TODO implement me
	panic("implement me")
}

type jsonResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

func (config *apiConfig) handleJSON(writer http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)

	jsonMesg := jsonBody{}
	err := decoder.Decode(&jsonMesg)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		writer.WriteHeader(500)
		return
	}

	jsonResp, err := validateJsonBody(jsonMesg.Body)
	if err != nil {
		dat, err := json.Marshal(err)
		if err != nil {
			log.Printf("json.Marshal error: %s", err)
			writer.WriteHeader(500)
			return
		}
		writer.WriteHeader(400)
		writer.Write(dat)
		return
	}

	response := jsonResponse{
		CleanedBody: jsonResp,
	}

	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("json.Marshal error: %s", err)
		writer.WriteHeader(500)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	writer.Write(dat)
}

func validateJsonBody(body string) (response string, err error) {
	if len(body) >= 140 {
		response = ""
		err = jsonError{
			Err: "Body is too long",
		}
		return
	}

	response = replaceProfanity(body)
	return
}

var profaneWords = map[string]bool{
	"kerfuffle": true,
	"sharbert":  true,
	"fornax":    true,
}

func replaceProfanity(text string) (res string) {
	words := strings.Fields(text)
	censored := "****"
	for i, _ := range words {
		if _, ok := profaneWords[strings.ToLower(words[i])]; ok {
			words[i] = censored
		}
	}

	res = strings.Join(words, " ")
	return
}

func (config *apiConfig) createUser(writer http.ResponseWriter, req *http.Request) {
	type jsonEmail struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)

	jsonMesg := jsonEmail{}
	err := decoder.Decode(&jsonMesg)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		writer.WriteHeader(500)
		return
	}

	dbUsr, err := config.dbQueries.CreateUser(req.Context(), jsonMesg.Email)
	if err != nil {
		log.Printf("db CreateUser failed: %s", err)
		writer.WriteHeader(500)
		return
	}

	user := User{
		ID:        dbUsr.ID,
		CreatedAt: dbUsr.CreatedAt,
		UpdatedAt: dbUsr.UpdatedAt,
		Email:     dbUsr.Email,
	}

	dat, err := json.Marshal(user)
	if err != nil {
		log.Printf("json.Marshal error: %s", err)
		writer.WriteHeader(500)
		return
	}

	writer.WriteHeader(201)
	writer.Write(dat)
}
