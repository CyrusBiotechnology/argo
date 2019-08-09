package gcs

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/argoproj/pkg/s3"
	"github.com/cyrusbiotechnology/argo/util"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	log "github.com/sirupsen/logrus"
)

// This is modeled heavily on https://github.com/argoproj/pkg/blob/master/s3/s3.go and
// Should probably be moved into argoproj/pkg at some point

// S3Client is a totally reasonable interface, using an alias so the typenames make sense
type GCSClient s3.S3Client

type GCSClientOpts struct {
	CredsJSONData []byte
}

type gcsClient struct {
	GCSClientOpts
	context context.Context
	client *storage.Client
}


type uploadTask struct {
	key  string
	path string
}


func NewGCSClient(opts GCSClientOpts) (client GCSClient, err error) {
	gcs := gcsClient{
		GCSClientOpts: opts,
	}

	gcs.context = context.Background()

	gcs.client, err = storage.NewClient(gcs.context, option.WithCredentialsJSON(gcs.CredsJSONData))
	if err != nil {
		return
	}
	return
}


//plagiarized from github.com/argoproj/pkg/s3
func generatePutTasks(keyPrefix, rootPath string) chan uploadTask {
	rootPath = filepath.Clean(rootPath) + "/"
	uploadTasks := make(chan uploadTask)
	visit := func(localPath string, fi os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(localPath, rootPath)
		if fi.IsDir() {
			return nil
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		t := uploadTask{
			key:  path.Join(keyPrefix, relPath),
			path: localPath,
		}
		uploadTasks <- t
		return nil
	}
	go func() {
		_ = filepath.Walk(rootPath, visit)
		close(uploadTasks)
	}()
	return uploadTasks
}

func (g *gcsClient) PutFile(bucket, key, path string) error {
	inputFile, err := os.Open(path)
	if err != nil {
		return err
	}

	defer util.Close(inputFile)

	bucketHandle := g.client.Bucket(bucket)
	object := bucketHandle.Object(key)

	w := object.NewWriter(g.Context)
	_, err = io.Copy(w, inputFile)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}

func (g *gcsClient) PutDirectory(bucket, key, path string) error {
	for putTask := range generatePutTasks(key, path) {
		err := g.PutFile(bucket, putTask.key, putTask.path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gcsClient) GetFile(bucket, key, path string) error {
	log.Infof("Getting from GCS (bucket: %s, key: %s) to %s", bucket, key, path)
	outputFile, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	bucketHandle := g.client.Bucket(bucket)
	object := bucketHandle.Object(key)

	r, err := object.NewReader(g.Context)
	if err != nil {
		return err
	}
	defer util.Close(r)

	_, err = io.Copy(outputFile, r)
	if err != nil {
		return err
	}

	err = outputFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func (g *gcsClient) GetDirectory(bucket, keyPrefix, path string) error {
	log.Infof("Getting directory from gcs (bucket: %s, key: %s) to %s", bucket, keyPrefix, path)
	bucketHandle := g.client.Bucket(bucket)
	it := bucketHandle.Objects(g.Context, &storage.Query{Prefix: keyPrefix})
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		localPath := filepath.Join(path, objAttrs.Name)
		err =  g.GetFile(bucket, objAttrs.Name, localPath)
		if err != nil {
			return err
		}
	}
	return nil


}

func (g *gcsClient) IsDirectory(bucket, key string) (bool, error) {
	bucketHandle := g.client.Bucket(bucket)
	// If the item in the query result has a name that matches the key, this is a file not a directory
	it := bucketHandle.Objects(g.Context, &storage.Query{Prefix: key})
	objectAttrs,  err := it.Next()
	if err != nil {
		return false, err
	}
	if objectAttrs.Name == key {
		return false, nil
	}
	return true, nil

}

