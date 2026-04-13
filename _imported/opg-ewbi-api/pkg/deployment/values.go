package deployment

import "github.com/neonephos-katalis/opg-ewbi-api/api/federation/server"

type Values struct {
	FederationContextID string                                 `json:"federationContextId,omitempty"`
	File                *server.ViewFile200JSONResponse        `json:"file,omitempty"`
	Artefact            *server.GetArtefact200JSONResponse     `json:"artefact,omitempty"`
	Application         *server.ViewApplication200JSONResponse `json:"application,omitempty"`
	ApplicationInstance *AppInstanceValues                     `json:"applicationInstance,omitempty"`
}

type AppInstanceValues struct {
	AppInstanceID string `json:"appInstanceId"`
	ZoneID        string `json:"zoneId"`
}
