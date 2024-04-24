package mediautils

import (
	"fmt"

	"github.com/zihaolam/golang-media-upload-server/internal"
)

func GetAbsolutePath(path string) string {
	return fmt.Sprintf("%s/%s", internal.Env.PublicAssetEndpoint, path)
}
