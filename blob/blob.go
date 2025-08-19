package blob

import (
	"context"
	"fmt"
	"mailtoblob/config"
	"mailtoblob/logger"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

const timeout = 180

func UploadFileToAzureBlobStorage(config *config.AzureConfig, address *string, msgBody *string, objectKey string, prefix string) error {
	// Calculating folder and subfolder
	currentDate := time.Now().UTC()
	//dateFolder := currentDate.Format("02-01-2006")
	hourFolder := currentDate.Format("15")
	minute := currentDate.Minute()

	var quarter string
	switch {
	case minute < 15:
		quarter = "00"
	case minute < 30:
		quarter = "15"
	case minute < 45:
		quarter = "30"
	default:
		quarter = "45"
	}

	// Folder structure: <prefix>/<date>/<hour>/<15min>/<objectKey>
	//blobName := path.Join(prefix, dateFolder, hourFolder, quarter, objectKey)
	blobName := path.Join(prefix, hourFolder, quarter, objectKey)

	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		logger.Log.Printf("[ERROR] %s", err)
		return fmt.Errorf("failed to create shared key credential: %w", err)
	}

	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, err := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", config.AccountName, config.ContainerName, blobName))
	if err != nil {
		logger.Log.Printf("[ERROR] %s", err)
		return fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Create a context with a timeout that will abort the upload if it takes
	// more than the passed in timeout.
	ctx := context.Background()
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancelFn()

	blockBlobURL := azblob.NewBlockBlobURL(*URL, pipeline)
	reader := strings.NewReader(*msgBody)
	_, err = azblob.UploadStreamToBlockBlob(ctx, reader, blockBlobURL, azblob.UploadStreamToBlockBlobOptions{})
	if err != nil {
		logger.Log.Printf("[ERROR] %s", err)
		return fmt.Errorf("failed to upload data to blob: %w", err)
	}
	return nil
}
