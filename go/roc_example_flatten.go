// Compare to examples/roc_example_flatten.c

package main
import (
    "fmt"
    "os"
)

// #cgo LDFLAGS: -lroc
// #include <roc.h>
import "C"

func main() {
    if len(os.Args) != 2 {
        C.roc_ensure(C.CString("Expected one image path argument:\n" +
                               "    $ go run roc_example_flatten.go path/to/image.jpg"))
    }

    // Initialize SDK
    C.roc_ensure(C.roc_initialize(nil, nil))

    // Open the image
    var image C.roc_image
    C.roc_ensure(C.roc_read_image(C.CString(os.Args[1]), C.ROC_GRAY8, &image))

    // Find and represent one face in the image
    var adaptive_minimum_size C.size_t
    var template_ C.roc_template
    C.roc_ensure(C.roc_adaptive_minimum_size(image, 0.08, 36, &adaptive_minimum_size))
    C.roc_ensure(C.roc_represent(image, C.ROC_FRONTAL | C.ROC_FR, adaptive_minimum_size, 1, 0.02, &template_))
    if (template_.algorithm_id & C.ROC_INVALID != 0) {
        fmt.Println("Failed to detect face in image:", os.Args[1])
        os.Exit(C.EXIT_FAILURE)
    }

    // Flatten the template to a buffer
    var buffer_size C.size_t
    C.roc_ensure(C.roc_flattened_bytes(template_, &buffer_size))
    buffer := make([]byte, buffer_size)
    C.roc_ensure(C.roc_flatten(template_, (*C.uint8_t)(&buffer[0])))
    fmt.Println("Flattened size:", buffer_size, "bytes")

    // Unflatten the template from a buffer
    var template_copy C.roc_template
    C.roc_ensure(C.roc_unflatten((*C.uint8_t)(&buffer[0]), &template_copy));

    // Cleanup
    C.roc_ensure(C.roc_free_template(&template_));
    C.roc_ensure(C.roc_free_template(&template_copy));
    C.roc_ensure(C.roc_free_image(image));
    C.roc_ensure(C.roc_finalize());
}
