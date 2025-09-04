package maxbot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type uploads struct {
	client *client
}

func newUploads(client *client) *uploads {
	return &uploads{client: client}
}

// UploadMedia uploads file to Max server
func (a *uploads) UploadMediaFromFile(ctx context.Context, uploadType schemes.UploadType, filename string) (*schemes.UploadedInfo, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	result := new(schemes.UploadedInfo)
	return result, a.uploadMediaFromReaderWithName(ctx, uploadType, fh, result, filepath.Base(filename))
}

// UploadMediaFromUrl uploads file from remote server to Max server
func (a *uploads) UploadMediaFromUrl(ctx context.Context, uploadType schemes.UploadType, u url.URL) (*schemes.UploadedInfo, error) {
	respFile, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer respFile.Body.Close()
	return a.UploadMediaFromReader(ctx, uploadType, respFile.Body)
}

func (a *uploads) UploadMediaFromReader(ctx context.Context, uploadType schemes.UploadType, reader io.Reader) (*schemes.UploadedInfo, error) {
	result := new(schemes.UploadedInfo)
	return result, a.uploadMediaFromReaderWithName(ctx, uploadType, reader, result, "file")
}

// UploadPhotoFromFile uploads photos to Max server
func (a *uploads) UploadPhotoFromFile(ctx context.Context, fileName string) (*schemes.PhotoTokens, error) {
	fh, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	result := new(schemes.PhotoTokens)
	return result, a.uploadMediaFromReaderWithName(ctx, schemes.PHOTO, fh, result, filepath.Base(fileName))
}

// UploadPhotoFromFile uploads photos to Max server
func (a *uploads) UploadPhotoFromBase64String(ctx context.Context, code string) (*schemes.PhotoTokens, error) {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(code))
	result := new(schemes.PhotoTokens)
	return result, a.uploadMediaFromReaderWithName(ctx, schemes.PHOTO, decoder, result, "file")
}

// UploadPhotoFromUrl uploads photo from remote server to Max server
func (a *uploads) UploadPhotoFromUrl(ctx context.Context, url string) (*schemes.PhotoTokens, error) {
	respFile, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer respFile.Body.Close()
	result := new(schemes.PhotoTokens)
	return result, a.uploadMediaFromReaderWithName(ctx, schemes.PHOTO, respFile.Body, result, "file")
}

// UploadPhotoFromReader uploads photo from reader
func (a *uploads) UploadPhotoFromReader(ctx context.Context, reader io.Reader) (*schemes.PhotoTokens, error) {
	result := new(schemes.PhotoTokens)
	return result, a.uploadMediaFromReaderWithName(ctx, schemes.PHOTO, reader, result, "file")
}

func (a *uploads) getUploadURL(ctx context.Context, uploadType schemes.UploadType) (*schemes.UploadEndpoint, error) {
	result := new(schemes.UploadEndpoint)
	values := url.Values{}
	values.Set("type", string(uploadType))
	body, err := a.client.request(ctx, http.MethodPost, "uploads", values, false, nil)
	if err != nil {
		return result, err
	}
	defer func() {
		if err := body.Close(); err != nil {
			log.Println(err)
		}
	}()
	return result, json.NewDecoder(body).Decode(result)
}

func (a *uploads) uploadMediaFromReaderWithName(ctx context.Context, uploadType schemes.UploadType, reader io.Reader, result interface{}, filename string) error {
	endpoint, err := a.getUploadURL(ctx, uploadType)
	if err != nil {
		return err
	}
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("data", filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	if err := bodyWriter.Close(); err != nil {
		return err
	}

	resp, err := http.Post(endpoint.Url, contentType, bodyBuf)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}
