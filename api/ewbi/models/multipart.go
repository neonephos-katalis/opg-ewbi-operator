package models

import (
	"encoding/json"
	"errors"

	"github.com/labstack/echo/v4"
)

func NewUploadArtefactMultipartBody(c echo.Context) (*UploadArtefactMultipartBody, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	// Ugly way to create the object, but I couldn't find a better way. So for now this is fine.
	// We are sure that the item [0] exists, otherwise the validator would fail.
	body := &UploadArtefactMultipartBody{
		AppProviderId:          form.Value["appProviderId"][0],
		ArtefactDescriptorType: (ArtefactDescriptorType)(form.Value["artefactDescriptorType"][0]),
		ArtefactId:             ArtefactId(form.Value["artefactId"][0]),
		ArtefactName:           form.Value["artefactName"][0],
		ArtefactVersionInfo:    form.Value["artefactVersionInfo"][0],
		ArtefactVirtType:       (ArtefactVirtType)(form.Value["artefactVirtType"][0]),
	}

	if vals, ok := form.Value["artefactDescription"]; ok && len(vals) > 0 {
		body.ArtefactDescription = &vals[0]
	}
	if vals, ok := form.Value["artefactNotifLink"]; ok && len(vals) > 0 {
		link := Uri(vals[0])
		body.ArtefactNotifLink = &link
	}

	if err := json.Unmarshal([]byte(form.Value["componentSpec"][0]), &body.ComponentSpec); err != nil {
		return nil, err
	}

	// Optional parameters
	// ArtefactFile *openapi_types.File `json:"artefactFile,omitempty"`
	// ArtefactFileFormat *UploadArtefactMultipartBodyArtefactFileFormat `json:"artefactFileFormat,omitempty"`
	// RepoType:            (UploadArtefactMultipartBodyRepoType)(form.Value["repoType"][0]),

	if vals, ok := form.Value["artefactFileName"]; ok && len(vals) > 0 {
		body.ArtefactFileName = &vals[0]
	}
	if vals, ok := form.Value["artefactRepoLocation"]; ok && len(vals) > 0 {
		if err := json.Unmarshal([]byte(vals[0]), &body.ArtefactRepoLocation); err != nil {
			return nil, err
		}
	}

	return body, nil
}

func NewUploadFileMultipartBody(c echo.Context) (*UploadFileMultipartBody, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	// Required parameters
	body := &UploadFileMultipartBody{
		AppProviderId:   AppProviderId(form.Value["appProviderId"][0]),
		FileId:          FileId(form.Value["fileId"][0]),
		FileName:        FileName(form.Value["fileName"][0]),
		FileType:        VirtImageType(form.Value["fileType"][0]),
		FileVersionInfo: FileVersionInfo(form.Value["fileVersionInfo"][0]),
		ImgInsSetArch:   CPUArchType(form.Value["imgInsSetArch"][0]),
	}

	// Optional parameters
	if vals, ok := form.Value["fileNotifLink"]; ok && len(vals) > 0 {
		link := Uri(vals[0])
		body.FileNotifLink = &link
	}
	if vals, ok := form.Value["repoType"]; ok && len(vals) > 0 {
		rt := RepoType(vals[0])
		body.RepoType = &rt
	}
	if vals, ok := form.Value["fileRepoLocation"]; ok && len(vals) > 0 {
		if err := json.Unmarshal([]byte(vals[0]), &body.FileRepoLocation); err != nil {
			return nil, err
		}
	}
	if vals, ok := form.Value["imgOSType"]; ok && len(vals) > 0 {
		if err := json.Unmarshal([]byte(vals[0]), &body.ImgOSType); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("missing required field imgOSType")
	}

	return body, nil
}
