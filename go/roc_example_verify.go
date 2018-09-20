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

// #cgo LDFLAGS: -lroc
// #include <roc.h>
import "C"

// InvalidSimilarity value for similarity when verification failed for some reason
const InvalidSimilarity = -1.0

type verificationResult struct {
	Similarity C.roc_similarity `json:"similarity"`
	Code       string           `json:"code"`
	Message    string           `json:"message"`
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
		C.roc_ensure(C.CString("Expected one argument: port"))
	}

	if port, err = strconv.Atoi(os.Args[1]); err != nil {
		C.roc_ensure(C.CString("Expected port to be a number"))
	}

	// init SDK
	C.roc_ensure(C.roc_initialize(nil, nil))

	r := mux.NewRouter()
	r.Methods("POST")
	r.Schemes("http")
	r.HandleFunc("/verify", verifyHandler)

	// http.Handle("/", r)
	http.ListenAndServe(fmt.Sprintf("localhost:%d", port), r)

	// for testing
	if len(os.Args) > 2 {
		log.Println("Checking image path", os.Args[1], os.Args[2])
		filePaths := [2]string{
			os.Args[1],
			os.Args[2],
		}

		var result = verify(filePaths)
		if result.Similarity == InvalidSimilarity {
			log.Panic(result.Message)
		} else {
			log.Println("result:", result)
		}
	}

	// wait for close
	defer func() {
		log.Println("cleanup")
		// cleanup SDK
		C.roc_ensure(C.roc_finalize())
	}()
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	var filePaths, err = saveImagesFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponseObj{
			Message: err.Error(),
		})

		return
	}

	var result = verify(filePaths)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func genTmpPath() string {
	return "./tmp/" + strconv.Itoa(rand.Int())
}

func saveImagesFromRequest(r *http.Request) ([2]string, error) {
	//ParseMultipartForm parses a request body as multipart/form-data
	r.ParseMultipartForm(32 << 20)

	formFields := [2]string{"image1", "image2"}
	var filePaths [2]string
	var formImages [2]multipart.File

	for index, field := range formFields {
		image, _, err := r.FormFile(field)
		defer image.Close() //close the file when we finish
		if err != nil {
			return filePaths, err
		}

		formImages[index] = image
	}

	for index, image := range formImages {
		var imagePath = genTmpPath()
		file, err := os.OpenFile(imagePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return filePaths, err
		}

		log.Printf("wrote file: %s", imagePath)
		filePaths[index] = imagePath
		defer file.Close()
		io.Copy(file, image)
	}

	return filePaths, nil
}

func verify(filePaths [2]string) verificationResult {
	// if len(filePaths) != 2 {
	// 	return verificationResult{
	// 		Similarity: InvalidSimilarity,
	// 		Code:       "InvalidImageCount",
	// 		Message:    "expected two image paths",
	// 	}
	// }

	// Open both images
	var images [2]C.roc_image
	for i := 0; i < 2; i++ {
		log.Println("Checking image path", filePaths[i])
		C.roc_ensure(C.roc_read_image(C.CString(filePaths[i]), C.ROC_GRAY8, &images[i]))
	}

	// Find and represent one face in each image
	var templates [2]C.roc_template
	for i := 0; i < 2; i++ {
		var adaptiveMinimumSize C.size_t
		C.roc_ensure(C.roc_adaptive_minimum_size(images[i], 0.08, 36, &adaptiveMinimumSize))
		C.roc_ensure(C.roc_represent(images[i], C.ROC_FRONTAL|C.ROC_FR, adaptiveMinimumSize, 1, 0.02, &templates[i]))
		if templates[i].algorithm_id&C.ROC_INVALID != 0 {
			var message = fmt.Sprintf("Failed to detect face in image %d", i)
			log.Println(message)
			return verificationResult{
				Similarity: InvalidSimilarity,
				Code:       "FaceNotDetected",
				Message:    message,
			}
		}
	}

	// Compare faces
	var similarity C.roc_similarity
	C.roc_ensure(C.roc_compare_templates(templates[0], templates[1], &similarity))
	log.Println("Similarity:", similarity)

	// Cleanup
	for i := 0; i < 2; i++ {
		C.roc_ensure(C.roc_free_template(&templates[i]))
		C.roc_ensure(C.roc_free_image(images[i]))
	}

	deleteFiles(filePaths[:])
	return verificationResult{
		Similarity: similarity,
	}
}
