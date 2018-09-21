// Compare to examples/roc_example_verify.c

package main
import (
    "fmt"
    "os"
)

// #cgo LDFLAGS: -lroc
// #include <roc.h>
import "C"

func main() {
    if len(os.Args) != 3 {
        C.roc_ensure(C.CString("Expected two image path arguments:\n" +
                               "    $ go run roc_example_verify.go path/to/image_a.jpg path/to/image_b.jpg"))
    }

    // Initialize SDK
    C.roc_ensure(C.roc_initialize(nil, nil))

    // Open both images
    var images [2]C.roc_image
    for i:=0; i<2; i++ {
        C.roc_ensure(C.roc_read_image(C.CString(os.Args[i+1]), C.ROC_GRAY8, &images[i]))
    }

    // Find and represent one face in each image
    var templates [2]C.roc_template
    for i:=0; i<2; i++ {
        var adaptive_minimum_size C.size_t
        C.roc_ensure(C.roc_adaptive_minimum_size(images[i], 0.08, 36, &adaptive_minimum_size))
        C.roc_ensure(C.roc_represent(images[i], C.ROC_FRONTAL | C.ROC_FR, adaptive_minimum_size, 1, 0.02, &templates[i]))
        if (templates[i].algorithm_id & C.ROC_INVALID != 0) {
            fmt.Println("Failed to detect face in image:", os.Args[i+1])
            os.Exit(C.EXIT_FAILURE)
        }
    }

    // Compare faces
    var similarity C.roc_similarity
    C.roc_ensure(C.roc_compare_templates(templates[0], templates[1], &similarity))
    fmt.Println("Similarity:", similarity)

    // Cleanup
    for i:=0; i<2; i++ {
        C.roc_ensure(C.roc_free_template(&templates[i]))
        C.roc_ensure(C.roc_free_image(images[i]))
    }
    C.roc_ensure(C.roc_finalize())
}

