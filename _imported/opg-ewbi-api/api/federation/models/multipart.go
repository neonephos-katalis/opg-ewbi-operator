package models

import (
	"encoding/json"

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
		ArtefactDescriptorType: (UploadArtefactMultipartBodyArtefactDescriptorType)(form.Value["artefactDescriptorType"][0]),
		ArtefactId:             ArtefactId(form.Value["artefactId"][0]),
		ArtefactName:           form.Value["artefactName"][0],
		ArtefactVersionInfo:    form.Value["artefactVersionInfo"][0],
		ArtefactVirtType:       (UploadArtefactMultipartBodyArtefactVirtType)(form.Value["artefactVirtType"][0]),
	}

	if err := json.Unmarshal([]byte(form.Value["componentSpec"][0]), &body.ComponentSpec); err != nil {
		return nil, err
	}

	// Optional parameters
	// ArtefactDescription *string `json:"artefactDescription,omitempty"`
	// ArtefactFile *openapi_types.File `json:"artefactFile,omitempty"`
	// ArtefactFileFormat *UploadArtefactMultipartBodyArtefactFileFormat `json:"artefactFileFormat,omitempty"`
	// RepoType:            (UploadArtefactMultipartBodyRepoType)(form.Value["repoType"][0]),
	if len(form.Value["artefactFileName"]) != 0 {
		body.ArtefactFileName = &form.Value["artefactFileName"][0]
	}
	if len(form.Value["artefactRepoLocation"]) != 0 {
		if err := json.Unmarshal([]byte(form.Value["artefactRepoLocation"][0]), &body.ArtefactRepoLocation); err != nil {
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
	// Left as a reminder
	// file := openapi_types.File{}
	// if err := ourecho.BindFromFile(c, "file", &file); err != nil {
	// 	return nil, err
	// }
	// Ugly way to create the object, but I couldn't find a better way. So for now this is fine.
	// We are sure that the item [0] exists, otherwise the validator would fail.
	body := &UploadFileMultipartBody{
		AppProviderId: form.Value["appProviderId"][0],
		// Checksum *string `json:"checksum,omitempty"`
		// File: &file,
		// FileDescription *string `json:"fileDescription,omitempty"`
		FileId:          FileId(form.Value["fileId"][0]),
		FileName:        form.Value["fileName"][0],
		FileType:        (VirtImageType)(form.Value["fileType"][0]),
		FileVersionInfo: form.Value["fileVersionInfo"][0],

		ImgInsSetArch: (CPUArchType)(form.Value["imgInsSetArch"][0]),

		RepoType: (*UploadFileMultipartBodyRepoType)(&form.Value["repoType"][0]),
	}

	if err := json.Unmarshal([]byte(form.Value["fileRepoLocation"][0]), &body.FileRepoLocation); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(form.Value["imgOSType"][0]), &body.ImgOSType); err != nil {
		return nil, err
	}

	return body, nil
}
