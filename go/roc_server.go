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
		C.roc_ensure(C.CString("Expected one argument: port"))
	}

	if port, err = strconv.Atoi(os.Args[1]); err != nil {
		C.roc_ensure(C.CString("Expected port to be a number"))
	}

	// init SDK
	log.Println("inializing sdk")
	C.roc_ensure(C.roc_initialize(nil, nil))
	log.Println("inialized sdk")

	// if len(os.Args) > 2 {
	// 	var filePaths = [2]string{os.Args[2], os.Args[3]}
	// 	log.Println("Checking image paths", filePaths)
	// 	var result = verify(filePaths)
	// 	if result.Similarity == InvalidSimilarity {
	// 		log.Panic(result.Message)
	// 	} else {
	// 		log.Println("result:", result)
	// 	}
	// }

	r := mux.NewRouter()
	r.Schemes("http")
	r.HandleFunc("/verify", verifyHandler).Methods("POST")

	var host = fmt.Sprintf("localhost:%d", port)
	log.Printf("running server on: %s", host)
	http.Handle("/", r)

	// wait for close
	defer func() {
		log.Println("cleanup")
		// cleanup SDK
		C.roc_ensure(C.roc_finalize())
	}()

	err = http.ListenAndServe(host, r)
	log.Println("the serveri is dead")
	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
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

	var result = verify(filePaths)
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

func verify(filePaths [2]string) verificationResult {
	// if len(filePaths) != 2 {
	// 	return verificationResult{
	// 		Similarity: InvalidSimilarity,
	// 		Code:       "InvalidImageCount",
	// 		Message:    "expected two image paths",
	// 	}
	// }

	log.Println("verify()")

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

	return verificationResult{
		Similarity: float32(similarity),
	}
}
