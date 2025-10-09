package multipart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	opgmodels "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/models"
)

type MultipartReader struct {
	writer *multipart.Writer
}

func NewMultipartReader(body *bytes.Buffer) *MultipartReader {
	return &MultipartReader{writer: multipart.NewWriter(body)}
}

// SerializeUploadFileMultipartBody serializes the struct to io.Reader for multipart/form-data
func SerializeUploadFileMultipartBody(fileMPBody opgmodels.UploadFileMultipartBody) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	fileReader := NewMultipartReader(body)

	// Add form fields from the struct
	if err := fileReader.addFormField("appProviderId", fileMPBody.AppProviderId); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormFieldPtr("checksum", fileMPBody.Checksum); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormFieldPtr("fileDescription", fileMPBody.FileDescription); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormField("fileId", fileMPBody.FileId); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormField("fileName", fileMPBody.FileName); err != nil {
		return nil, "", err
	}
	if err := fileReader.addObjectRepoLocationFormField("fileRepoLocation", fileMPBody.FileRepoLocation); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormField("fileType", string(fileMPBody.FileType)); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormField("fileVersionInfo", fileMPBody.FileVersionInfo); err != nil {
		return nil, "", err
	}
	if err := fileReader.addFormField("imgInsSetArch", string(fileMPBody.ImgInsSetArch)); err != nil {
		return nil, "", err
	}
	if err := fileReader.addOSTypeFormField("imgOSType", fileMPBody.ImgOSType); err != nil {
		return nil, "", err
	}
	if fileMPBody.RepoType != nil { // Handle potential nil pointer
		if err := fileReader.addFormField("repoType", string(*fileMPBody.RepoType)); err != nil {
			return nil, "", err
		}
	}

	err := fileReader.close() // Important: Close the writer to finalize the multipart body
	if err != nil {
		return nil, "", err
	}

	contentType := fileReader.formDataContentType()
	return body, contentType, nil
}

// SerializeUploadArtefactMultipartBody serializes the struct to io.Reader for multipart/form-data
func SerializeUploadArtefactMultipartBody(aMPBody opgmodels.UploadArtefactMultipartBody) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	aReader := NewMultipartReader(body)

	// Add form fields from the struct
	if err := aReader.addFormField("appProviderId", aMPBody.AppProviderId); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormFieldPtr("artefactDescription", aMPBody.ArtefactDescription); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormField("artefactId", aMPBody.ArtefactId); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormField("artefactName", aMPBody.ArtefactName); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormField("artefactVersionInfo", aMPBody.ArtefactVersionInfo); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormField("artefactVirtType", string(aMPBody.ArtefactVirtType)); err != nil {
		return nil, "", err
	}
	if err := aReader.addFormField("artefactDescriptorType", string(aMPBody.ArtefactDescriptorType)); err != nil {
		return nil, "", err
	}
	if err := aReader.addComponentSpecField("componentSpec", aMPBody.ComponentSpec); err != nil {
		return nil, "", err
	}

	err := aReader.close() // Important: Close the writer to finalize the multipart body
	if err != nil {
		return nil, "", err
	}

	contentType := aReader.formDataContentType()
	return body, contentType, nil
}

func (f MultipartReader) close() error {
	return f.writer.Close()
}

func (f MultipartReader) formDataContentType() string {
	return f.writer.FormDataContentType()
}

func (f MultipartReader) addFormField(fieldName string, fieldValue string) error {
	if fieldValue != "" { // Only add if value is not empty (or handle nil pointers appropriately)
		err := f.writer.WriteField(fieldName, fieldValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// Helper function to add form field from OSType struct
func (f MultipartReader) addComponentSpecField(fieldName string, comps []opgmodels.ComponentSpec) error {
	if len(comps) > 0 {
		componentSpecJSON, err := json.Marshal(comps)
		if err != nil {
			return err
		}
		err = f.writer.WriteField(fieldName, string(componentSpecJSON))
		if err != nil {
			return err
		}
	}
	return nil
}

// Helper function to add form field from OSType struct
func (f MultipartReader) addOSTypeFormField(fieldName string, osType opgmodels.OSType) error {
	if osType != (opgmodels.OSType{}) {
		osTypeJSON, err := json.Marshal(osType)
		if err != nil {
			return err
		}
		err = f.writer.WriteField(fieldName, string(osTypeJSON))
		if err != nil {
			return err
		}
	}
	return nil
}

// Helper function to add form field from ObjectRepoLocation struct
// notice the API SPEC expects the param value to be a json !!!
func (f MultipartReader) addObjectRepoLocationFormField(
	fieldName string,
	repoLocation *opgmodels.ObjectRepoLocation,
) error {
	if repoLocation != (&opgmodels.ObjectRepoLocation{}) {
		repoLocationJSON, err := json.Marshal(repoLocation)
		if err != nil {
			return err
		}
		err = f.writer.WriteField(fieldName, string(repoLocationJSON))
		if err != nil {
			return err
		}
	}
	return nil
}

// Helper function to add form field from pointer to string
func (f MultipartReader) addFormFieldPtr(fieldName string, fieldValuePtr *string) error {
	if fieldValuePtr != nil && *fieldValuePtr != "" {
		return f.addFormField(fieldName, *fieldValuePtr)
	}
	return nil
}

func GetFormFieldValueFromReader(multipartReader io.Reader, contentTypeHeader string, fieldName string) (string, error) {
	// Parse the Content-Type header to get the boundary.
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return "", fmt.Errorf("error parsing Content-Type header: %w", err)
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return "", fmt.Errorf("invalid Content-Type: not a multipart type")
	}
	boundary := params["boundary"]
	if boundary == "" {
		return "", fmt.Errorf("boundary not found in Content-Type header")
	}

	mr := multipart.NewReader(multipartReader, boundary)

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break // No more parts
		}
		if err != nil {
			return "", fmt.Errorf("error reading next multipart part: %w", err)
		}
		defer part.Close()

		disposition := part.Header.Get("Content-Disposition")
		if disposition == "" {
			continue
		}

		_, dispParams, err := mime.ParseMediaType(disposition)
		if err != nil {
			continue // Could not parse Content-Disposition, skip part
		}

		name := dispParams["name"]
		if name == fieldName {
			valueBytes, err := io.ReadAll(part)
			if err != nil {
				return "", fmt.Errorf("error reading part body for field '%s': %w", fieldName, err)
			}
			return string(valueBytes), nil
		}
	}

	return "", fmt.Errorf("field '%s' not found in multipart data", fieldName)
}
