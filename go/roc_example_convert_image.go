// Illustrates how to convert a roc_image to and from a Go image

package main
import (
    "image"
//    "image/png"
    "os"
    "unsafe"
)

// #cgo LDFLAGS: -lroc
// #include <roc.h>
import "C"

func main() {
    if len(os.Args) != 2 {
        C.roc_ensure(C.CString("Expected one image path argument:\n" +
                               "    $ go run roc_example_convert_image.go path/to/image.jpg"))
    }

    // Initialize SDK
    C.roc_ensure(C.roc_initialize(nil, nil))

    // Open the input image
    var img C.roc_image
    C.roc_ensure(C.roc_read_image(C.CString(os.Args[1]), C.ROC_BGR24, &img))

    // Convert from a roc_image to a Go image
    img_go := image.NewRGBA(image.Rect(0, 0, int(img.width), int(img.height)))
    img_data := C.GoBytes(unsafe.Pointer(img.data), C.int(3 * img.width * img.height))
    for i:=C.size_t(0); i<img.height; i++ {
        for j:=C.size_t(0); j<img.width; j++ {
            srcIndex := int(i*img.step + 3*j)
            dstIndex := int(i*C.size_t(img_go.Stride) + 4*j)
            img_go.Pix[dstIndex + 0] = img_data[srcIndex + 2] // R
            img_go.Pix[dstIndex + 1] = img_data[srcIndex + 1] // G
            img_go.Pix[dstIndex + 2] = img_data[srcIndex + 0] // B
            img_go.Pix[dstIndex + 3] = 255 // A
        }
    }

    // out, _ := os.Create("./img_go.png")
    // png.Encode(out, img_go)

    // Convert from a Go image to a roc_image
    var img_roc C.roc_image
    img_roc.width = C.size_t(img_go.Rect.Dx())
    img_roc.height = C.size_t(img_go.Rect.Dy())
    img_roc.step = C.size_t(3 * img_go.Rect.Dy())
    img_roc.color_space = C.ROC_BGR24
    img_roc_data := make([]byte, 3 * img_roc.width * img_roc.height);
    img_roc.data = (*C.uint8_t)(&img_roc_data[0]);

    for i:=C.size_t(0); i<img.height; i++ {
        for j:=C.size_t(0); j<img.width; j++ {
            srcIndex := int(i*C.size_t(img_go.Stride) + 4*j)
            dstIndex := int(i*img_roc.step + 3*j)
            img_roc_data[dstIndex + 0] = img_go.Pix[srcIndex + 2] // B
            img_roc_data[dstIndex + 1] = img_go.Pix[srcIndex + 1] // G
            img_roc_data[dstIndex + 2] = img_go.Pix[srcIndex + 0] // R
        }
    }

    // C.roc_ensure(C.roc_write_image(img_roc, C.CString("img_roc.png")))

    // Cleanup
    C.roc_ensure(C.roc_free_image(img))
    C.roc_ensure(C.roc_finalize())
}
