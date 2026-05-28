package deployment

import (
	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
)

type InstallDeployment struct {
	*models.InstallAppJSONBody
	FederationContextID string
}
