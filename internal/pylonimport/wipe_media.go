package pylonimport

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
)

// deleteStoredMessageMedia removes message attachment blobs from the upload store (and image thumbs).
func deleteStoredMessageMedia(ctx context.Context, mgr *media.Manager, out io.Writer, files []mmodels.Media) (deleted, deleteErrors int, err error) {
	if len(files) == 0 {
		return 0, 0, nil
	}
	if mgr == nil {
		return 0, 0, fmt.Errorf("a media manager is required to delete message attachments")
	}
	for _, mm := range files {
		if ctx.Err() != nil {
			return deleted, deleteErrors, ctx.Err()
		}
		if err := mgr.Delete(mm.UUID); err != nil {
			deleteErrors++
			fmt.Fprintf(out, "warning: delete media %s: %v\n", mm.UUID, err)
			continue
		}
		deleted++
		if strings.HasPrefix(mm.ContentType, "image/") {
			thumb := image.ThumbPrefix + mm.UUID
			if err := mgr.Delete(thumb); err != nil {
				fmt.Fprintf(out, "warning: delete thumb %s: %v\n", thumb, err)
			}
		}
	}
	return deleted, deleteErrors, nil
}
