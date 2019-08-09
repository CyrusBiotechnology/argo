package gcs

import (
	"github.com/argoproj/pkg/file"
	argoErrors "github.com/cyrusbiotechnology/argo/errors"
	wfv1 "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cyrusbiotechnology/argo/util"
	log "github.com/sirupsen/logrus"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"time"
)

type GCSArtifactDriver struct {
	CredsJSONData []byte
}

func (gcsDriver *GCSArtifactDriver) newGcsClient() (client GCSClient, err error) {

	client, err = NewGCSClient(GCSClientOpts{CredsJSONData: gcsDriver.CredsJSONData})

	if err != nil {
		return nil, argoErrors.InternalWrapError(err)
	}
	return

}

func (gcsDriver *GCSArtifactDriver) saveToFile(inputArtifact *wfv1.Artifact, filePath string) error {

	log.Infof("Loading from GCS (gs://%s/%s) to %s",
		inputArtifact.GCS.Bucket, inputArtifact.GCS.Key, filePath)

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

	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	//if stat.IsDir() {
	//	for putTask := range generatePutTasks(outputArtifact.GCS.Bucket, outputArtifact.GCS.Key, filePath) {
	//		err :=
	//	}
	//	return errors.New("only single files can be saved to GCS, not entire directories")
	//}

	defer util.Close(inputFile)

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

	bucketName := inputArtifact.GCS.Bucket
	key := inputArtifact.GCS.Key

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("GCS Load path: %s, key: %s", path, key)
			gcsClient, err := gcsDriver.newGcsClient()
			if err != nil {
				log.Warnf("Failed to create new GCS client: %v", err)
				return false, nil
			}

			isDir, err := gcsClient.IsDirectory(bucketName, key)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", bucketName, err)
				return false, nil
			}

			if isDir {
				if err = gcsClient.GetDirectory(bucketName, key, path)
				err != nil {
					log.Warnf("Failed get directory: %v", err)
					return false, nil
				}
			} else {
				err := gcsClient.GetFile(bucketName, key, path)
				if  err != nil {
					return false, nil
				}
			}

			return true, nil
		})

	return err
}

func (gcsDriver *GCSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	bucketName := outputArtifact.GCS.Bucket
	key := outputArtifact.GCS.Key

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, key)
			gcsClient, err := gcsDriver.newGcsClient()
			if err != nil {
				log.Warnf("Failed to create new S3 client: %v", err)
				return false, nil
			}
			isDir, err := file.IsDirectory(path)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", path, err)
				return false, nil
			}
			if isDir {
				if err = gcsClient.PutDirectory(bucketName, key, path); err != nil {
					log.Warnf("Failed to put directory: %v", err)
					return false, nil
				}
			} else {
				if err = gcsClient.PutFile(bucketName, key, path); err != nil {
					log.Warnf("Failed to put file: %v", err)
					return false, nil
				}
			}
			return true, nil
		})
	return err
}
