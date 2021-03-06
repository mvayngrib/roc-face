// Compare to examples/roc_example_verify.c

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	// Third party packages
	"github.com/gorilla/mux"
)

// InvalidSimilarity value for similarity when verification failed for some reason
const InvalidSimilarity = -1.0
const _32M = (1 << 20) * 32

type verificationResult struct {
	Similarity float32 `json:"similarity,omitempty"`
	Code       string  `json:"code,omitempty"`
	Message    string  `json:"message,omitempty"`
}

func deleteFiles(filePaths []string) {
	log.Println("deleting files:", filePaths)
	for _, filePath := range filePaths {
		os.Remove(filePath)
	}
}

type errorResponseObj struct {
	Message string `json:"message"`
}

func main() {
	var port int
	var err error
	if len(os.Args) < 2 {
		log.Fatal("Expected one argument: port")
	}

	if port, err = strconv.Atoi(os.Args[1]); err != nil {
		log.Fatal("Expected port to be a number")
	}

	// init SDK
	log.Println("inializing sdk")
	log.Println("inialized sdk")

	r := mux.NewRouter()
	r.Schemes("http")
	r.HandleFunc("/verify", verifyHandler).Methods("POST")
	r.HandleFunc("/ping", pingHandler).Methods("GET", "POST")

	var host = fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("running server on: %s", host)
	http.Handle("/", r)

	// wait for close
	defer func() {
		log.Println("cleanup")
		// cleanup SDK
	}()

	err = http.ListenAndServe(host, r)
	log.Println("the server is dead")
	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/ping")
	w.WriteHeader(http.StatusOK)
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/verify")
	var filePaths, err = saveImagesFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponseObj{
			Message: err.Error(),
		})

		return
	}

	var result = verificationResult{
		Similarity: rand.Float32(),
	}

	deleteFiles(filePaths[:])

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func genTmpPath() string {
	var tmpPath = "/tmp/roc-face-" + strconv.Itoa(rand.Int())
	log.Println("generated tmp path", tmpPath)
	return tmpPath
}

func saveImagesFromRequest(r *http.Request) ([2]string, error) {
	var err error

	//ParseMultipartForm parses a request body as multipart/form-data
	r.ParseMultipartForm(_32M) // max 16MB images

	log.Println("extracting images from request")
	formFields := [2]string{"image1", "image2"}
	var filePaths [2]string
	var formImages [2]multipart.File

	for index, field := range formFields {
		log.Println("extracting field", field)
		image, _, err := r.FormFile(field)
		if err != nil {
			log.Println("failed to extract field:", field, "error:", err.Error())
			return filePaths, err
		}

		defer image.Close() //close the file when we finish
		formImages[index] = image
	}

	log.Println("saving images to disk")
	for index, image := range formImages {
		var imagePath = genTmpPath()
		var outfile *os.File
		outfile, err = os.Create(imagePath)
		if err != nil {
			return filePaths, err
		}

		_, err = io.Copy(outfile, image)
		if err != nil {
			return filePaths, err
		}

		filePaths[index] = imagePath
		defer outfile.Close()
		io.Copy(outfile, image)
		log.Printf("wrote file: %s", imagePath)
	}

	return filePaths, nil
}
