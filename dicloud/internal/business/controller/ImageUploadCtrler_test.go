package controller

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func maintainTestImage(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func maintainTestDirectory(t *testing.T) string {
	t.Helper()
	previousDirectory, previousLimit := viper.Get("uploads.directory"), viper.Get("uploads.max_size_mb")
	directory := filepath.Join(t.TempDir(), "new-uploads")
	viper.Set("uploads.directory", directory)
	viper.Set("uploads.max_size_mb", 1)
	t.Cleanup(func() {
		viper.Set("uploads.directory", previousDirectory)
		viper.Set("uploads.max_size_mb", previousLimit)
	})
	return directory
}

func maintainUploadRequest(t *testing.T, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/go/images/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestMaintainImageUploadAndReadSameOrigin(t *testing.T) {
	directory := maintainTestDirectory(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/go/images/upload", Upload)
	router.GET("/go/images/file/:filename", ReadUploadedImage)
	data := maintainTestImage(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, maintainUploadRequest(t, "原始图片.png", data))
	var result struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || result.Code != 200 || !strings.HasPrefix(result.Data.URL, "/go/images/file/") {
		t.Fatalf("upload failed: %s, err=%v", recorder.Body.String(), err)
	}
	if _, err := os.Stat(filepath.Join(directory, filepath.Base(result.Data.URL))); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, result.Data.URL, nil))
	if recorder.Code != http.StatusOK || !bytes.Equal(data, recorder.Body.Bytes()) || recorder.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("image read failed: status=%d", recorder.Code)
	}
	// 旧域名仅用于提取同名本地文件，测试不会产生外部请求。
	legacy := "https://old.invalid/uploads/" + filepath.Base(result.Data.URL)
	loaded, extension, err := readMaintainImage(legacy)
	if err != nil || extension != ".png" || !bytes.Equal(data, loaded) {
		t.Fatalf("legacy local lookup failed: extension=%s err=%v", extension, err)
	}
	details, _ := json.Marshal(map[string]string{"imageUrl": legacy})
	workbook, err := buildMaintainWorkbook([]dao.MaintainRecordRow{{
		EquipmentMaintainRecord: model.EquipmentMaintainRecord{MaintainDetails: details},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	pictures, err := workbook.GetPictures("Sheet1", "D2")
	if err != nil || len(pictures) != 1 {
		t.Fatalf("workbook does not contain uploaded image: count=%d err=%v", len(pictures), err)
	}
}

func TestMaintainImageRejectsInvalidUploadAndUnsafeNames(t *testing.T) {
	directory := maintainTestDirectory(t)
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"fake.png", []byte("<html>not an image</html>")},
		{"fake.jpg", maintainTestImage(t)},
		{"large.png", make([]byte, (1<<20)+1)},
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = maintainUploadRequest(t, test.name, test.data)
		Upload(ctx)
		var result map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &result)
		if result["code"] != float64(400) {
			t.Fatalf("invalid upload accepted: %s", recorder.Body.String())
		}
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("invalid upload unexpectedly created storage directory")
	}
	for _, filename := range []string{"../secret.png", `..\secret.png`, "C:secret.png", ".hidden.png", "file.svg"} {
		if file, err := openMaintainImage(filename); err == nil {
			file.Close()
			t.Fatalf("unsafe filename accepted: %q", filename)
		}
	}
}
