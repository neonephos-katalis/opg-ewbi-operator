package metastore

const (
	opgLabelKeyPrefix = "opg.ewbi.katalis.com"
)

type labelKey string

const (
	clientIDLabel             labelKey = "origin-client-id"
	federationCallbackIDLabel labelKey = "federation-callback-id"
	federationContextIDLabel  labelKey = "federation-context-id"
	federationRelation        labelKey = "federation-relation"
	idLabel                   labelKey = "id"
	kindLabel                 labelKey = "kind"
)

const (
	applicationDeploymentKind   string = "applicationDeployment"
	applicationDeploymentPrefix string = "application-deployment"
	applicationOnboardingKind   string = "applicationOnboarding"
	artefactKind                string = "artefact"
	availabilityZoneKind        string = "availabilityZone"
	federationKind              string = "federation"
	imageKind                   string = "image"
)

const (
	// values for federation relation label
	host  string = "HOST"
	guest string = "GUEST"
)

func opgLabel(l labelKey) string {
	return opgLabelKeyPrefix + "/" + string(l)
}
