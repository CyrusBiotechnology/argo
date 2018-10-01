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

func (gcsDriver *GCSArtifactDriver) saveToFile(inputArtifact *wfv1.Artifact, filePath string) error {

	log.Infof("Loading from GCS (gs://%s/%s) to %s",
		inputArtifact.GCS.Bucket, inputArtifact.GCS.Key, filePath)

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

	gcsClient, err := gcsDriver.newGcsClient()
	if err != nil {
		return err
	}

	bucket := gcsClient.Bucket(inputArtifact.GCS.Bucket)
	object := bucket.Object(inputArtifact.GCS.Key)

	r, err := object.NewReader(gcsDriver.Context)
	if err != nil {
		return err
	}
	defer r.Close()

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

func (gcsDriver *GCSArtifactDriver) saveToGCS(outputArtifact *wfv1.Artifact, filePath string) error {

	log.Infof("Saving to GCS (gs://%s/%s)",
		outputArtifact.GCS.Bucket, outputArtifact.GCS.Key)

	gcsClient, err := gcsDriver.newGcsClient()
	if err != nil {
		return err
	}

	inputFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	bucket := gcsClient.Bucket(outputArtifact.GCS.Bucket)
	object := bucket.Object(outputArtifact.GCS.Key)

	w := object.NewWriter(gcsDriver.Context)
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

func (gcsDriver *GCSArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {

	err := gcsDriver.saveToFile(inputArtifact, path)
	if err != nil {
		return err
	}

	return nil
}

func (gcsDriver *GCSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {

	err := gcsDriver.saveToGCS(outputArtifact, path)
	if err != nil {
		return err
	}

	return nil

}
