package deployment

import (
	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

type InstallDeployment struct {
	*models.InstallAppJSONBody
	FederationContextID string
}
