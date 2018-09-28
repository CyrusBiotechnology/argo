package gcs

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	argoErrors "github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type GCSArtifactDriver struct {
	Context context.Context
}

func (gcsDriver *GCSArtifactDriver) newGcsClient() (client *storage.Client, err error) {
	gcsDriver.Context = context.Background()
	client, err = storage.NewClient(gcsDriver.Context)
	if err != nil {
		return nil, argoErrors.InternalWrapError(err)
	}
	return

}

func writeToFile(r *storage.Reader, filePath string) error {
	stat, err := os.Stat(filePath)
	if err == nil {
		if stat.IsDir() {
			return errors.New("output artifact path is a directory")
		}
	}

	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	outputFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	_, err = io.Copy(outputFile, r)
	if err != nil {
		return err
	}
	outputFile.Close()
	return nil
}

func readFromFile(w *storage.Writer, filePath string) error {
	inputFile, err := os.Open(filePath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, inputFile)
	if err != nil {
		return err
	}

	inputFile.Close()
	return nil

}

func (gcsDriver *GCSArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	gcsClient, err := gcsDriver.newGcsClient()
	if err != nil {
		return err
	}

	bucket := gcsClient.Bucket(inputArtifact.GCS.Bucket)
	log.Infof("Loading from GCS (bucket: %s, key: %s) to %s",
		inputArtifact.GCS.Bucket, inputArtifact.GCS.Key, path)

	object := bucket.Object(inputArtifact.GCS.Key)
	r, err := object.NewReader(gcsDriver.Context)
	if err != nil {
		return err
	}
	err = writeToFile(r, path)
	if err != nil {
		return err
	}
	return nil
}

func (gcsDriver *GCSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	gcsClient, err := gcsDriver.newGcsClient()
	if err != nil {
		return err
	}

	log.Infof("Loading from GCS (bucket: %s, key: %s)",
		outputArtifact.GCS.Bucket, outputArtifact.GCS.Key)

	bucket := gcsClient.Bucket(outputArtifact.GCS.Bucket)
	object := bucket.Object(outputArtifact.GCS.Key)
	w := object.NewWriter(gcsDriver.Context)

	err = readFromFile(w, path)
	if err != nil {
		return err
	}
	return nil

}
