package upload

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"wageslavemonitor/client/internal/spool"
)

type Uploader struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func New(baseURL, token string) *Uploader {
	return &Uploader{
		BaseURL: baseURL,
		Token:   token,
		Client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (u *Uploader) Upload(ctx context.Context, item spool.Item) error {
	img, err := os.ReadFile(item.ImagePath)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("monitor_index", strconv.Itoa(item.MonitorIndex))
	_ = writer.WriteField("captured_at", item.CapturedAt.UTC().Format(time.RFC3339))
	part, err := writer.CreateFormFile("image", "screen.jpg")
	if err != nil {
		return err
	}
	if _, err := part.Write(img); err != nil {
		return err
	}
	_ = writer.Close()

	url := fmt.Sprintf("%s/api/v1/clients/%s/screenshots", u.BaseURL, item.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if u.Token != "" {
		req.Header.Set("Authorization", "Bearer "+u.Token)
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("upload failed: %s", resp.Status)
	}
	return nil
}
