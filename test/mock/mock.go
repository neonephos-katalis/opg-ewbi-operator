package mock

import (
	"context"
	"io"
	"net/http"

	opgewbiv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"

	opgc "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/client"
	"github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/models"
	opgmodels "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/models"

	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/multipart"
)

const (
	alreadyExistsMsg  = "conflict object already exists"
	notImplementedMsg = "not implemented"
)

type MockedOpgAPI struct {
	Federations map[string]*opgewbiv1beta1.Federation
	Files       map[string]*opgewbiv1beta1.File
	Artefacts   map[string]*opgewbiv1beta1.Artefact
	Apps        map[string]*opgewbiv1beta1.Application
	AppInsts    map[string]*opgewbiv1beta1.ApplicationInstance
	AZs         map[string]*opgewbiv1beta1.AvailabilityZone
}

func MakeMokedOpgAPI() *MockedOpgAPI {
	c := &MockedOpgAPI{
		Federations: make(map[string]*opgewbiv1beta1.Federation),
		Files:       make(map[string]*opgewbiv1beta1.File),
		Artefacts:   make(map[string]*opgewbiv1beta1.Artefact),
		Apps:        make(map[string]*opgewbiv1beta1.Application),
		AppInsts:    make(map[string]*opgewbiv1beta1.ApplicationInstance),
		AZs:         make(map[string]*opgewbiv1beta1.AvailabilityZone),
	}
	return c
}

func (c *MockedOpgAPI) WithFederations(feds []*opgewbiv1beta1.Federation) *MockedOpgAPI {
	for _, f := range feds {
		// the index in this function is different than the others, because in this case,
		// the client's function only has the federation-context-id and not the federation-external-id
		c.Federations[f.Status.FederationContextId] = f
	}
	return c
}

func (c *MockedOpgAPI) WithFiles(files []*opgewbiv1beta1.File) *MockedOpgAPI {
	for _, f := range files {
		c.Files[f.Labels[opgewbiv1beta1.ExternalIdLabel]] = f
	}
	return c
}

func (c *MockedOpgAPI) WithArtefacts(arts []*opgewbiv1beta1.Artefact) *MockedOpgAPI {
	for _, a := range arts {
		c.Artefacts[a.Labels[opgewbiv1beta1.ExternalIdLabel]] = a
	}
	return c
}

func (c *MockedOpgAPI) WithApplications(apps []*opgewbiv1beta1.Application) *MockedOpgAPI {
	for _, a := range apps {
		c.Apps[a.Labels[opgewbiv1beta1.ExternalIdLabel]] = a
	}
	return c
}

func (c *MockedOpgAPI) WithApplicationInstances(appInsts []*opgewbiv1beta1.ApplicationInstance) *MockedOpgAPI {
	for _, a := range appInsts {
		c.AppInsts[a.Labels[opgewbiv1beta1.ExternalIdLabel]] = a
	}
	return c
}

func (c *MockedOpgAPI) WithAZs(azs []*opgewbiv1beta1.AvailabilityZone) *MockedOpgAPI {
	for _, a := range azs {
		c.AZs[a.Labels[opgewbiv1beta1.ExternalIdLabel]] = a
	}
	return c
}

func (c *MockedOpgAPI) CreateFederationWithResponse(
	ctx context.Context,
	body opgmodels.CreateFederationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.CreateFederationResponse, error) {

	var res *opgc.CreateFederationResponse
	// federation must already be pre-registered, otherwise the partner-guest can't join it
	f, ok := c.Federations[body.OrigOPFederationId]
	if ok {
		// we assume creds are valid
		// we should update the fed with the new data
		// c.Federations[body.OrigOPFederationId] = ...
		// return AZs

		res = &opgc.CreateFederationResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &opgmodels.FederationResponseData{
				OfferedAvailabilityZones: &[]opgmodels.ZoneDetails{{ZoneId: f.Status.OfferedAvailabilityZones[0].ZoneId}},
				FederationContextId:      &f.Status.FederationContextId,
			},
		}

	} else {
		detail := "conflict federation doesn't exist"
		res = &opgc.CreateFederationResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	}

	return res, nil
}

// DeleteFederationDetailsWithResponse request returning *DeleteFederationDetailsResponse
func (c *MockedOpgAPI) DeleteFederationDetailsWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.DeleteFederationDetailsResponse, error) {

	var res *opgc.DeleteFederationDetailsResponse
	// if federation already exists return conflict
	_, ok := c.Federations[federationContextId]
	if !ok {
		detail := "unable to remove federation, not found"
		res = &opgc.DeleteFederationDetailsResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.Federations, federationContextId)
		res = &opgc.DeleteFederationDetailsResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil

}

// UploadFileWithBodyWithResponse request with arbitrary body returning *UploadFileResponse
func (c *MockedOpgAPI) UploadFileWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UploadFileResponse, error) {

	var res *opgc.UploadFileResponse

	fileName, err := multipart.GetFormFieldValueFromReader(body, contentType, "fileId")
	if err != nil {
		return nil, err
	}

	// if file already exists return conflict
	_, ok := c.Files[fileName]
	if ok {
		detail := alreadyExistsMsg
		res = &opgc.UploadFileResponse{
			HTTPResponse: &http.Response{
				StatusCode: 409,
			},
			ApplicationproblemJSON409: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		// for now we are not storing it... we are using the map as a set
		c.Files[fileName] = &opgewbiv1beta1.File{}
		res = &opgc.UploadFileResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil

}

// RemoveFileWithResponse request returning *RemoveFileResponse
func (c *MockedOpgAPI) RemoveFileWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	fileId opgmodels.FileId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.RemoveFileResponse, error) {

	var res *opgc.RemoveFileResponse
	// if file already exists return conflict
	_, ok := c.Files[fileId]
	if !ok {
		detail := "unable to remove file, not found"
		res = &opgc.RemoveFileResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.Files, fileId)
		res = &opgc.RemoveFileResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil

}

// UploadArtefactWithBodyWithResponse request with arbitrary body returning *UploadArtefactResponse
func (c *MockedOpgAPI) UploadArtefactWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UploadArtefactResponse, error) {
	var res *opgc.UploadArtefactResponse

	aName, err := multipart.GetFormFieldValueFromReader(body, contentType, "artefactId")
	if err != nil {
		return nil, err
	}

	// if artefact already exists return conflict
	_, ok := c.Artefacts[aName]
	if ok {
		detail := alreadyExistsMsg
		res = &opgc.UploadArtefactResponse{
			HTTPResponse: &http.Response{
				StatusCode: 409,
			},
			ApplicationproblemJSON409: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		// for now we are not storing it... we are using the map as a set
		c.Artefacts[aName] = &opgewbiv1beta1.Artefact{}
		res = &opgc.UploadArtefactResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

// RemoveArtefactWithResponse request returning *RemoveArtefactResponse
func (c *MockedOpgAPI) RemoveArtefactWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	artefactId opgmodels.ArtefactId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.RemoveArtefactResponse, error) {

	var res *opgc.RemoveArtefactResponse
	// if artefact already exists return conflict
	_, ok := c.Artefacts[artefactId]
	if !ok {
		detail := "unable to remove artefact, not found"
		res = &opgc.RemoveArtefactResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.Artefacts, artefactId)
		res = &opgc.RemoveArtefactResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

func (c *MockedOpgAPI) OnboardApplicationWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.OnboardApplicationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.OnboardApplicationResponse, error) {
	var res *opgc.OnboardApplicationResponse

	aName := body.AppId

	// if app already exists return conflict
	_, ok := c.Apps[aName]
	if ok {
		detail := alreadyExistsMsg
		res = &opgc.OnboardApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: 409,
			},
			ApplicationproblemJSON409: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		// for now we are not storing it... we are using the map as a set
		c.Apps[aName] = &opgewbiv1beta1.Application{}
		res = &opgc.OnboardApplicationResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

// DeleteAppWithResponse request returning *DeleteAppResponse
func (c *MockedOpgAPI) DeleteAppWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.DeleteAppResponse, error) {
	var res *opgc.DeleteAppResponse
	// if app already exists return conflict
	_, ok := c.Apps[appId]
	if !ok {
		detail := "unable to remove app, not found"
		res = &opgc.DeleteAppResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.Apps, appId)
		res = &opgc.DeleteAppResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

func (c *MockedOpgAPI) InstallAppWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.InstallAppJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.InstallAppResponse, error) {
	var res *opgc.InstallAppResponse

	aName := body.AppInstanceId

	// if appInst already exists return conflict
	_, ok := c.AppInsts[aName]
	if ok {
		detail := alreadyExistsMsg
		res = &opgc.InstallAppResponse{
			HTTPResponse: &http.Response{
				StatusCode: 409,
			},
			ApplicationproblemJSON409: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		// for now we are not storing it... we are using the map as a set
		c.AppInsts[aName] = &opgewbiv1beta1.ApplicationInstance{}
		res = &opgc.InstallAppResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

// RemoveAppWithResponse request returning *RemoveAppResponse
func (c *MockedOpgAPI) RemoveAppWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appInstanceId opgmodels.InstanceIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.RemoveAppResponse, error) {
	var res *opgc.RemoveAppResponse
	// if appInst already exists return conflict
	_, ok := c.AppInsts[appInstanceId]
	if !ok {
		detail := "unable to remove appInst, not found"
		res = &opgc.RemoveAppResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.AppInsts, appInstanceId)
		res = &opgc.RemoveAppResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

func (c *MockedOpgAPI) ZoneSubscribeWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.ZoneSubscribeJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ZoneSubscribeResponse, error) {

	aName := body.AcceptedAvailabilityZones[0]

	var res *opgc.ZoneSubscribeResponse
	// if az already exists return conflict
	_, ok := c.AZs[aName]
	if ok {
		detail := alreadyExistsMsg
		res = &opgc.ZoneSubscribeResponse{
			HTTPResponse: &http.Response{
				StatusCode: 409,
			},
			ApplicationproblemJSON409: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		// for now we are not storing it... we are using the map as a set
		c.AZs[aName] = &opgewbiv1beta1.AvailabilityZone{}
		res = &opgc.ZoneSubscribeResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

// ZoneUnsubscribeWithResponse request returning *ZoneUnsubscribeResponse
func (c *MockedOpgAPI) ZoneUnsubscribeWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ZoneUnsubscribeResponse, error) {
	var res *opgc.ZoneUnsubscribeResponse
	// if appInst already exists return conflict
	_, ok := c.AZs[zoneId]
	if !ok {
		detail := "unable to remove AZ, not found"
		res = &opgc.ZoneUnsubscribeResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			ApplicationproblemJSON404: &opgmodels.ProblemDetails{
				Detail: &detail,
			},
		}
	} else {
		delete(c.AZs, zoneId)
		res = &opgc.ZoneUnsubscribeResponse{
			Body: []byte{},
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
		}
	}

	return res, nil
}

// mocks not yet implemented

// CreateFederationWithBodyWithResponse request with arbitrary body returning *CreateFederationResponse
func (c *MockedOpgAPI) CreateFederationWithBodyWithResponse(
	ctx context.Context,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.CreateFederationResponse, error) {
	panic(notImplementedMsg)
}

// InstallAppWithBodyWithResponse request with arbitrary body returning *opgc.InstallAppResponse
func (c *MockedOpgAPI) InstallAppWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.InstallAppResponse, error) {
	panic(notImplementedMsg)
}

// GetAllAppInstancesWithResponse request returning *GetAllAppInstancesResponse
func (c *MockedOpgAPI) GetAllAppInstancesWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appProviderId opgmodels.AppProviderId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetAllAppInstancesResponse, error) {
	panic(notImplementedMsg)
}

// GetAppInstanceDetailsWithResponse request returning *GetAppInstanceDetailsResponse
func (c *MockedOpgAPI) GetAppInstanceDetailsWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appInstanceId opgmodels.InstanceIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetAppInstanceDetailsResponse, error) {
	panic(notImplementedMsg)
}

// OnboardApplicationWithBodyWithResponse request with arbitrary body returning *OnboardApplicationResponse
func (c *MockedOpgAPI) OnboardApplicationWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.OnboardApplicationResponse, error) {
	panic(notImplementedMsg)
}

// ViewApplicationWithResponse request returning *ViewApplicationResponse
func (c *MockedOpgAPI) ViewApplicationWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ViewApplicationResponse, error) {
	panic(notImplementedMsg)
}

// UpdateApplicationWithBodyWithResponse request with arbitrary body returning *UpdateApplicationResponse
func (c *MockedOpgAPI) UpdateApplicationWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateApplicationResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateApplicationWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.UpdateApplicationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateApplicationResponse, error) {
	panic(notImplementedMsg)
}

// OnboardExistingAppNewZonesWithBodyWithResponse request with arbitrary body
// returning *OnboardExistingAppNewZonesResponse
func (c *MockedOpgAPI) OnboardExistingAppNewZonesWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.OnboardExistingAppNewZonesResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) OnboardExistingAppNewZonesWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.OnboardExistingAppNewZonesJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.OnboardExistingAppNewZonesResponse, error) {
	panic(notImplementedMsg)
}

// DeboardApplicationWithResponse request returning *DeboardApplicationResponse
func (c *MockedOpgAPI) DeboardApplicationWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.DeboardApplicationResponse, error) {
	panic(notImplementedMsg)
}

// LockUnlockApplicationZoneWithBodyWithResponse request with arbitrary body
// returning *LockUnlockApplicationZoneResponse
func (c *MockedOpgAPI) LockUnlockApplicationZoneWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.LockUnlockApplicationZoneResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) LockUnlockApplicationZoneWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.LockUnlockApplicationZoneJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.LockUnlockApplicationZoneResponse, error) {
	panic(notImplementedMsg)
}

// GetArtefactWithResponse request returning *GetArtefactResponse
func (c *MockedOpgAPI) GetArtefactWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	artefactId opgmodels.ArtefactId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetArtefactResponse, error) {
	panic(notImplementedMsg)
}

// GetCandidateZonesWithBodyWithResponse request with arbitrary body returning *GetCandidateZonesResponse
func (c *MockedOpgAPI) GetCandidateZonesWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetCandidateZonesResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetCandidateZonesWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.GetCandidateZonesJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetCandidateZonesResponse, error) {
	panic(notImplementedMsg)
}

// ViewFileWithResponse request returning *ViewFileResponse
func (c *MockedOpgAPI) ViewFileWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	fileId opgmodels.FileId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ViewFileResponse, error) {
	panic(notImplementedMsg)
}

// ViewISVResPoolWithResponse request returning *ViewISVResPoolResponse
func (c *MockedOpgAPI) ViewISVResPoolWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ViewISVResPoolResponse, error) {
	panic(notImplementedMsg)
}

// CreateResourcePoolsWithBodyWithResponse request with arbitrary body returning *CreateResourcePoolsResponse
func (c *MockedOpgAPI) CreateResourcePoolsWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.CreateResourcePoolsResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) CreateResourcePoolsWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	body opgmodels.CreateResourcePoolsJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.CreateResourcePoolsResponse, error) {
	panic(notImplementedMsg)
}

// RemoveISVResPoolWithResponse request returning *RemoveISVResPoolResponse
func (c *MockedOpgAPI) RemoveISVResPoolWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.RemoveISVResPoolResponse, error) {
	panic(notImplementedMsg)
}

// UpdateISVResPoolWithBodyWithResponse request with arbitrary body returning *UpdateISVResPoolResponse
func (c *MockedOpgAPI) UpdateISVResPoolWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateISVResPoolResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateISVResPoolWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	body opgmodels.UpdateISVResPoolJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateISVResPoolResponse, error) {
	panic(notImplementedMsg)
}

// GetFederationDetailsWithResponse request returning *GetFederationDetailsResponse
func (c *MockedOpgAPI) GetFederationDetailsWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetFederationDetailsResponse, error) {
	panic(notImplementedMsg)
}

// UpdateFederationWithBodyWithResponse request with arbitrary body returning *UpdateFederationResponse
func (c *MockedOpgAPI) UpdateFederationWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateFederationResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateFederationWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.UpdateFederationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.UpdateFederationResponse, error) {
	panic(notImplementedMsg)
}

// AuthenticateDeviceWithResponse request returning *AuthenticateDeviceResponse
func (c *MockedOpgAPI) AuthenticateDeviceWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	deviceId opgmodels.DeviceId,
	authToken opgmodels.AuthorizationToken,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AuthenticateDeviceResponse, error) {
	panic(notImplementedMsg)
}

// ZoneSubscribeWithBodyWithResponse request with arbitrary body returning *ZoneSubscribeResponse
func (c *MockedOpgAPI) ZoneSubscribeWithBodyWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ZoneSubscribeResponse, error) {
	panic(notImplementedMsg)
}

// GetZoneDataWithResponse request returning *GetZoneDataResponse
func (c *MockedOpgAPI) GetZoneDataWithResponse(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.GetZoneDataResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) CreateFederationWithBody(
	ctx context.Context,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) CreateFederation(
	ctx context.Context,
	body opgmodels.CreateFederationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) InstallAppWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) InstallApp(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.InstallAppJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetAllAppInstances(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appProviderId opgmodels.AppProviderId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) RemoveApp(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appInstanceId opgmodels.InstanceIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetAppInstanceDetails(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	appInstanceId opgmodels.InstanceIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) OnboardApplicationWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) OnboardApplication(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.OnboardApplicationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) DeleteApp(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ViewApplication(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateApplicationWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateApplication(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.UpdateApplicationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) OnboardExistingAppNewZonesWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) OnboardExistingAppNewZones(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.OnboardExistingAppNewZonesJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) DeboardApplication(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) LockUnlockApplicationZoneWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) LockUnlockApplicationZone(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	appId opgmodels.AppIdentifier,
	body opgmodels.LockUnlockApplicationZoneJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UploadArtefactWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) RemoveArtefact(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	artefactId opgmodels.ArtefactId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetArtefact(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	artefactId opgmodels.ArtefactId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetCandidateZonesWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetCandidateZones(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.GetCandidateZonesJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UploadFileWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) RemoveFile(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	fileId opgmodels.FileId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ViewFile(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	fileId opgmodels.FileId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ViewISVResPool(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) CreateResourcePoolsWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) CreateResourcePools(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	body opgmodels.CreateResourcePoolsJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) RemoveISVResPool(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateISVResPoolWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateISVResPool(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	appProviderId opgmodels.AppProviderId,
	poolId opgmodels.PoolId,
	body opgmodels.UpdateISVResPoolJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) DeleteFederationDetails(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetFederationDetails(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateFederationWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) UpdateFederation(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.UpdateFederationJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) AuthenticateDevice(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	deviceId opgmodels.DeviceId,
	authToken opgmodels.AuthorizationToken,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ZoneSubscribeWithBody(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ZoneSubscribe(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	body opgmodels.ZoneSubscribeJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ZoneUnsubscribe(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) GetZoneData(
	ctx context.Context,
	federationContextId opgmodels.FederationContextId,
	zoneId opgmodels.ZoneIdentifier,
	reqEditors ...opgc.RequestEditorFn,
) (*http.Response, error) {
	panic(notImplementedMsg)
}

// AppInstCallbackLinkWithBodyWithResponse request with arbitrary body returning *AppInstCallbackLinkResponse
func (c *MockedOpgAPI) AppInstCallbackLinkWithBodyWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AppInstCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) AppInstCallbackLinkWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	body models.AppInstCallbackLinkJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AppInstCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}

// AppStatusCallbackLinkWithBodyWithResponse request with arbitrary body returning *AppStatusCallbackLinkResponse
func (c *MockedOpgAPI) AppStatusCallbackLinkWithBodyWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AppStatusCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) AppStatusCallbackLinkWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	body models.AppStatusCallbackLinkJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AppStatusCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}

// AvailZoneNotifLinkWithBodyWithResponse request with arbitrary body returning *AvailZoneNotifLinkResponse
func (c *MockedOpgAPI) AvailZoneNotifLinkWithBodyWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AvailZoneNotifLinkResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) AvailZoneNotifLinkWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	body models.AvailZoneNotifLinkJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.AvailZoneNotifLinkResponse, error) {
	panic(notImplementedMsg)
}

// PartnerStatusLinkWithBodyWithResponse request with arbitrary body returning *PartnerStatusLinkResponse
func (c *MockedOpgAPI) PartnerStatusLinkWithBodyWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.PartnerStatusLinkResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) PartnerStatusLinkWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	body models.PartnerStatusLinkJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.PartnerStatusLinkResponse, error) {
	panic(notImplementedMsg)
}

// ResourceReservationCallbackLinkWithBodyWithResponse request with arbitrary body
// returning *ResourceReservationCallbackLinkResponse
func (c *MockedOpgAPI) ResourceReservationCallbackLinkWithBodyWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	contentType string,
	body io.Reader,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ResourceReservationCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}

func (c *MockedOpgAPI) ResourceReservationCallbackLinkWithResponse(
	ctx context.Context,
	federationCallbackId opgmodels.FederationCallbackId,
	body models.ResourceReservationCallbackLinkJSONRequestBody,
	reqEditors ...opgc.RequestEditorFn,
) (*opgc.ResourceReservationCallbackLinkResponse, error) {
	panic(notImplementedMsg)
}
