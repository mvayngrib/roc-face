// Compare to examples/roc_example_video.c

package main
import (
    "fmt"
    "os"
)

// #cgo LDFLAGS: -lroc -lroc_video
// #include <roc.h>
import "C"

func main() {
    if len(os.Args) != 2 {
        C.roc_ensure(C.CString("Expected one video path argument:\n" +
                               "    $ go run roc_example_video.go path/to/video.mp4"))
    }

    // Initialize SDK
    C.roc_ensure(C.roc_initialize(nil, nil))

    // Open the video
    var video C.roc_video
    var video_metadata C.roc_video_metadata
    C.roc_ensure(C.roc_open_video(C.CString(os.Args[1]), C.ROC_GRAY8, &video, &video_metadata))

    // Read the first frame
    var frame C.roc_image
    var timestamp C.roc_time
    C.roc_ensure(C.roc_read_frame(video, &frame, &timestamp))

    // Print video statistics
    fmt.Printf("Duration: %d ms\nWidth: %d pixels\nHeight: %d pixels\n", video_metadata.duration, frame.width, frame.height)

    // Cleanup
    C.roc_ensure(C.roc_close_video(video))
    C.roc_ensure(C.roc_free_image(frame))
    C.roc_ensure(C.roc_finalize())
}
