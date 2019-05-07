package gcs

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	argoErrors "github.com/cyrusbiotechnology/argo/errors"
	wfv1 "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cyrusbiotechnology/argo/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"time"
)

type GCSArtifactDriver struct {
	Context       context.Context
	CredsJSONData []byte
}

func (gcsDriver *GCSArtifactDriver) newGcsClient() (client *storage.Client, err error) {
	gcsDriver.Context = context.Background()

	client, err = storage.NewClient(gcsDriver.Context, option.WithCredentialsJSON(gcsDriver.CredsJSONData))
	if err != nil {
		return nil, argoErrors.InternalWrapError(err)
	}
	return

}

func (gcsDriver *GCSArtifactDriver) saveToFile(inputArtifact *wfv1.Artifact, filePath string) error {

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("Loading from GCS (gs://%s/%s) to %s",
				inputArtifact.GCS.Bucket, inputArtifact.GCS.Key, filePath)

			stat, err := os.Stat(filePath)
			if err != nil && !os.IsNotExist(err) {
				return false, err
			}

			if stat != nil && stat.IsDir() {
				return false, errors.New("output artifact path is a directory")
			}

			outputFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				return false, err
			}

			gcsClient, err := gcsDriver.newGcsClient()
			if err != nil {
				return false, err
			}

			bucket := gcsClient.Bucket(inputArtifact.GCS.Bucket)
			object := bucket.Object(inputArtifact.GCS.Key)

			r, err := object.NewReader(gcsDriver.Context)
			if err != nil {
				return false, err
			}
			defer util.Close(r)

			_, err = io.Copy(outputFile, r)
			if err != nil {
				return false, err
			}

			err = outputFile.Close()
			if err != nil {
				return false, err
			}

			return true, nil
		})

	return err
}

func (gcsDriver *GCSArtifactDriver) saveToGCS(outputArtifact *wfv1.Artifact, filePath string) error {

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("Saving to GCS (gs://%s/%s)",
				outputArtifact.GCS.Bucket, outputArtifact.GCS.Key)

			gcsClient, err := gcsDriver.newGcsClient()
			if err != nil {
				return false, err
			}

			inputFile, err := os.Open(filePath)
			if err != nil {
				return false, err
			}

			stat, err := os.Stat(filePath)
			if err != nil {
				return false, err
			}

			if stat.IsDir() {
				return false, errors.New("only single files can be saved to GCS, not entire directories")
			}

			defer util.Close(inputFile)

			bucket := gcsClient.Bucket(outputArtifact.GCS.Bucket)
			object := bucket.Object(outputArtifact.GCS.Key)

			w := object.NewWriter(gcsDriver.Context)
			_, err = io.Copy(w, inputFile)
			if err != nil {
				return false, err
			}

			err = w.Close()
			if err != nil {
				return false, err
			}
			return true, nil
		})

	return err

}

func (gcsDriver *GCSArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {

	err := gcsDriver.saveToFile(inputArtifact, path)
	return err
}

func (gcsDriver *GCSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {

	err := gcsDriver.saveToGCS(outputArtifact, path)
	return err
}
