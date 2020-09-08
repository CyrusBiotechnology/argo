package raw_test

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cyrusbiotechnology/argo/workflow/artifacts/raw"
)

const (
	LoadFileName string = "argo_raw_artifact_test_load.txt"
)

func TestLoad(t *testing.T) {

	content := "time: " + strconv.FormatInt(time.Now().UnixNano(), 10)
	lf, err := ioutil.TempFile("", LoadFileName)
	assert.Nil(t, err)
	defer os.Remove(lf.Name())

	art := &wfv1.Artifact{}
	art.Raw = &wfv1.RawArtifact{
		Data: content,
	}
	driver := &raw.RawArtifactDriver{}
	driver.Load(art, lf.Name())

	dat, err := ioutil.ReadFile(lf.Name())
	assert.Nil(t, err)
	assert.Equal(t, content, string(dat))

}
