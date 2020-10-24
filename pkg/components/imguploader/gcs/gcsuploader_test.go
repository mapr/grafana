package gcs

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/grafana/grafana/pkg/ifaces/gcsifaces"
	"github.com/grafana/grafana/pkg/mocks/mock_gcsifaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

const dfltExpiration = 7 * 24 * time.Hour

func TestUploadToGCS_DefaultCredentials(t *testing.T) {
	origNewClient := newClient
	t.Cleanup(func() {
		newClient = origNewClient
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const bucket = "test"
	content := []byte("test\n")
	ctx := context.Background()

	wm := mock_gcsifaces.NewMockStorageWriter(ctrl)
	wm.
		EXPECT().
		SetACL(gomock.Eq("publicRead")).
		Return()
	wm.EXPECT().
		Write(gomock.Eq(content)).
		Return(len(content), nil)
	wm.EXPECT().
		Close()

	om := mock_gcsifaces.NewMockStorageObject(ctrl)
	om.
		EXPECT().
		NewWriter(gomock.Eq(ctx)).
		Return(wm)

	bm := mock_gcsifaces.NewMockStorageBucket(ctrl)
	bm.
		EXPECT().
		Object(gomock.Any()).
		Return(om)

	cm := mock_gcsifaces.NewMockStorageClient(ctrl)
	cm.
		EXPECT().
		Bucket(gomock.Eq(bucket)).
		Return(bm)

	newClient = func(ctx context.Context, options ...option.ClientOption) (gcsifaces.StorageClient, error) {
		return cm, nil
	}

	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "test.png")
	err := ioutil.WriteFile(fpath, content, 0600)
	require.NoError(t, err)

	uploader, err := NewUploader("", bucket, "", false, dfltExpiration)
	require.NoError(t, err)

	path, err := uploader.Upload(ctx, fpath)
	require.NoError(t, err)

	assert.Regexp(t, fmt.Sprintf(`^https://storage.googleapis.com/%s/[^/]+\.png$`, bucket), path)
}
