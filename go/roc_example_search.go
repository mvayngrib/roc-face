// Compare to examples/roc_example_search.c

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
                               "    $ go run roc_example_search.go path/to/gallery.jpg path/to/probe.jpg"))
    }

    // Initialize SDK
    C.roc_ensure(C.roc_initialize(nil, nil))

    // Open both images
    var gallery_image, probe_image C.roc_image
    C.roc_ensure(C.roc_read_image(C.CString(os.Args[1]), C.ROC_GRAY8, &gallery_image))
    C.roc_ensure(C.roc_read_image(C.CString(os.Args[2]), C.ROC_GRAY8, &probe_image))

    // Construct gallery by finding all faces in the gallery image
    var adaptive_minimum_size C.size_t
    C.roc_ensure(C.roc_adaptive_minimum_size(gallery_image, 0.08, 36, &adaptive_minimum_size))
    const maximum_faces C.int = 10
    gallery_templates := make([]C.roc_template, maximum_faces)
    C.roc_ensure(C.roc_represent(gallery_image, C.ROC_FRONTAL | C.ROC_FR, adaptive_minimum_size, maximum_faces, 0.02, &gallery_templates[0]))

    var gallery C.roc_gallery
    C.roc_ensure(C.roc_open_gallery(nil, &gallery, nil))
    for i:=C.int(0); i<maximum_faces; i++ {
        if (gallery_templates[i].algorithm_id & C.ROC_INVALID != 0) {
            if i == 0 {
                C.roc_ensure(C.CString("Failed to find a face in the gallery image!"))
            }
            break
        }
        C.roc_ensure(C.roc_enroll(gallery, gallery_templates[i]))
    }

    // Find a single face in the probe image
    C.roc_ensure(C.roc_adaptive_minimum_size(probe_image, 0.08, 36, &adaptive_minimum_size))
    var probe C.roc_template
    C.roc_ensure(C.roc_represent(probe_image, C.ROC_FRONTAL | C.ROC_FR, adaptive_minimum_size, 1, 0.02, &probe))
    if (probe.algorithm_id & C.ROC_INVALID != 0) {
        C.roc_ensure(C.CString("Failed to find a face in the probe image!"))
    }

    // Print probe face quality
    var quality C.roc_string
    C.roc_ensure(C.roc_get_metadata(probe, C.CString("Quality"), &quality))
    fmt.Println("Probe Quality:", C.GoString(quality))
    C.roc_ensure(C.roc_free_string(&quality))

    // Execute search
    const maximum_candidates C.size_t = 3
    candidates := make([]C.roc_candidate, maximum_candidates)
    C.roc_ensure(C.roc_search(gallery, probe, maximum_candidates, 0.0, &candidates[0]))

    fmt.Println("Similarity\tX\tY\tWidth\tHeight");
    for i:=C.size_t(0); i<maximum_candidates; i++ {
        candidate := candidates[i]
        if (int(candidate.index) == -1 /* C.ROC_INVALID_TEMPLATE_INDEX */) {
            break
        }

        var candidate_template C.roc_template
        C.roc_ensure(C.roc_at(gallery, candidate.index, &candidate_template))
        fmt.Printf("%g\t%d\t%d\t%d\t%d\n", candidate.similarity, candidate_template.x, candidate_template.y, candidate_template.width, candidate_template.height)
        C.roc_ensure(C.roc_free_template(&candidate_template))
    }

    // Cleanup
    C.roc_ensure(C.roc_free_image(gallery_image))
    C.roc_ensure(C.roc_free_image(probe_image))
    C.roc_ensure(C.roc_close_gallery(gallery))
    for i:=C.int(0); i<maximum_faces; i++ {
        C.roc_ensure(C.roc_free_template(&gallery_templates[i]))
    }
    C.roc_ensure(C.roc_free_template(&probe))
    C.roc_ensure(C.roc_finalize())
}
