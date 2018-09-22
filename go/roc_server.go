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
const defaultFDR = 0.02
const defaultMinFaceWidthInPixels = 36
const defaultNumFacesToDetect = 1

var verifyFormFields = []string{"image1", "image2"}
var analyzeFormFields = []string{"image"}

type verificationResult struct {
	Similarity float32 `json:"similarity,omitempty"`
	Code       string  `json:"code,omitempty"`
	Message    string  `json:"message,omitempty"`
}

type analysisResult struct {
	Code                 string      `json:"code,omitempty"`
	Message              string      `json:"message,omitempty"`
	FDR                  float32     `json:"fdr,omitempty"`
	MinFaceWidthInPixels int         `json:"minFaceWidthInPixels,omitempty"`
	Analysis             interface{} `json:"analysis,omitempty"`
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

// func init() {
// 	log.Println("inializing sdk")
// 	C.roc_ensure(C.roc_initialize(nil, nil))
// 	log.Println("inialized sdk")
// }

func main() {
	var port int
	var err error

	log.Println("inializing sdk")
	C.roc_ensure(C.roc_initialize(nil, nil))
	log.Println("inialized sdk")

	var command = os.Args[1]
	if command == "verify" {
		if len(os.Args) < 5 {
			log.Fatal("expected image paths as next arguments")
		}

		var filePaths = []string{os.Args[2], os.Args[3]}
		log.Println("Checking image paths", filePaths)
		var result = verify(filePaths)
		if result.Similarity == InvalidSimilarity {
			log.Panic(result.Message)
		} else {
			log.Println("result:", result)
		}

		return
	}

	if command == "analyze" {
		if len(os.Args) < 3 {
			log.Fatal("expected image path as next argument")
		}

		var filePath = os.Args[2]
		log.Println("Analyzing image")
		var result = analyze(filePath, defaultFDR, defaultMinFaceWidthInPixels, defaultNumFacesToDetect)
		log.Println("Analysis:", result)

		return
	}

	if command != "serve" {
		log.Fatal("invalid command, expected one of: verify, analyze, serve")
	}

	if port, err = strconv.Atoi(os.Args[2]); err != nil {
		C.roc_ensure(C.CString("Expected port to be a number"))
	}

	r := mux.NewRouter()
	r.Schemes("http")
	r.HandleFunc("/verify", verifyHandler).Methods("POST")
	r.HandleFunc("/analyze", analyzeHandler).Methods("POST")

	var host = fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("running server on: %s", host)
	http.Handle("/", r)

	// wait for close
	defer func() {
		log.Println("cleanup")
		// cleanup SDK
		C.roc_ensure(C.roc_finalize())
	}()

	err = http.ListenAndServe(host, r)
	log.Println("the server is dead")
	if err != nil {
		log.Println("what killed it? I'm glad you asked:", err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func sendError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(errorResponseObj{
		Message: err.Error(),
	})
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/analyze")
	var filePaths, err = saveImagesFromRequest(r, analyzeFormFields)
	if err != nil {
		sendError(w, err)
		return
	}

	var fdr float32
	var minFaceWidthInPixels int
	var numFacesToDetect int
	fdr, err = getFloatQueryParam(r, "fdr", defaultFDR)
	if err != nil {
		sendError(w, err)
		return
	}

	minFaceWidthInPixels, err = getIntQueryParam(r, "minFaceWidthInPixels", defaultMinFaceWidthInPixels)
	if err != nil {
		sendError(w, err)
		return
	}

	numFacesToDetect, err = getIntQueryParam(r, "numFacesToDetect", defaultNumFacesToDetect)
	if err != nil {
		sendError(w, err)
		return
	}

	var result = analyze(filePaths[0], fdr, minFaceWidthInPixels, numFacesToDetect)
	deleteFiles(filePaths[:])

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func getIntQueryParam(r *http.Request, param string, defaultValue int) (int, error) {
	var val = r.URL.Query().Get(param)
	if len(val) > 0 {
		return strconv.Atoi(val)
	}

	return defaultValue, nil
}

func getFloatQueryParam(r *http.Request, param string, defaultValue float32) (float32, error) {
	var val = r.URL.Query().Get(param)
	if len(val) > 0 {
		var ret, err = strconv.ParseFloat(val, 32)
		if err != nil {
			return -1.0, err
		}

		return float32(ret), err
	}

	return defaultValue, nil
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("/verify")
	var filePaths, err = saveImagesFromRequest(r, verifyFormFields)
	if err != nil {
		// TODO: in case one image was succcessfully extracted, delete it

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

func saveImagesFromRequest(r *http.Request, formFields []string) ([]string, error) {
	var err error

	//ParseMultipartForm parses a request body as multipart/form-data
	r.ParseMultipartForm(_32M) // max 16MB images

	log.Println("extracting images from request")
	var filePaths = make([]string, len(formFields))
	var formImages = make([]multipart.File, len(formFields))

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

func analyze(filePath string, fdr float32, minFaceWidthInPixels int, numFacesToDetect int) analysisResult {
	log.Println("analyze()")
	var algorithmID C.roc_algorithm_id = C.ROC_FRONTAL | // detect frontal faces
		C.ROC_FR | // represent the face
		C.ROC_DEMOGRAPHICS | // extract demographics
		C.ROC_PITCHYAW | // extract face position information
		C.ROC_SPOOF_AF | // static image spoof detection
		C.ROC_GLASSES |
		C.ROC_LANDMARKS | // Add RightEyeX, RightEyeY, LeftEyeX, LeftEyeY, ChinX, ChinY, NoseRootX and NoseRootY pixel locations, and IOD (inter-occular pixel distance) to the template metadata.
		// C.ROC_THUMBNAIL |
		C.ROC_LIPS // lips apart vs together

	var image C.roc_image
	var template C.roc_template

	log.Println("Checking image path", filePath)
	C.roc_ensure(C.roc_read_image(C.CString(filePath), C.ROC_GRAY8, &image))

	log.Println("Analyzing face")
	C.roc_ensure(C.roc_represent(image, algorithmID, C.size_t(minFaceWidthInPixels), C.int(numFacesToDetect), C.float(fdr), &template))
	if template.algorithm_id&C.ROC_INVALID != 0 {
		var message = fmt.Sprintf("Failed to detect face in image")
		log.Println(message)
		return analysisResult{
			Code:    "FaceNotDetected",
			Message: message,
		}
	}

	var bytes = []byte(C.GoString(template.md))
	var data interface{}
	json.Unmarshal(bytes, &data)

	// for example:
	// {
	// 	"Age": 33,
	// 	"Asian": 0.0016529097920283675,
	// 	"Black": 0.00099143886473029852,
	// 	"ChinX": 537,
	// 	"ChinY": 373,
	// 	"Female": 0.0024212179705500603,
	// 	"Hispanic": 0.043269451707601547,
	// 	"IOD": 76,
	// 	"LeftEyeX": 580,
	// 	"LeftEyeY": 236,
	// 	"Male": 0.99757874011993408,
	// 	"NoseRootX": 544,
	// 	"NoseRootY": 224,
	// 	"Other": 0.0040792962536215782,
	// 	"Path": "",
	// 	"Pitch": 6,
	// 	"Pose": "Frontal",
	// 	"Quality": 0.45093154907226562,
	// 	"RightEyeX": 505,
	// 	"RightEyeY": 228,
	// 	"Roll": 2,
	// 	"SpoofAF": 0.67681902647018433,
	// 	"White": 0.95000690221786499,
	// 	"Yaw": -3
	// }

	// var similarity C.roc_similarity
	// C.roc_ensure(C.roc_(templates[0], templates[1], &similarity))
	// log.Println("Similarity:", similarity)

	// Cleanup
	C.roc_ensure(C.roc_free_template(&template))
	C.roc_ensure(C.roc_free_image(image))

	return analysisResult{
		FDR:                  fdr,
		MinFaceWidthInPixels: minFaceWidthInPixels,
		Analysis:             data,
	}
}

func verify(filePaths []string) verificationResult {
	if len(filePaths) != 2 {
		return verificationResult{
			Similarity: InvalidSimilarity,
			Code:       "InvalidImageCount",
			Message:    "expected two image paths",
		}
	}

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
