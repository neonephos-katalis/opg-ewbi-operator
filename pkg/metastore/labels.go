package metastore

const (
	opgLabelKeyPrefix = "opg.ewbi.nby.one"
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
	applicationInstanceKind   string = "applicationInstance"
	applicationInstancePrefix string = "application-instance"
	applicationKind           string = "application"
	artefactKind              string = "artefact"
	availabilityZoneKind      string = "availabilityZone"
	federationKind            string = "federation"
	fileKind                  string = "file"
)

const (
	// values for federation relation label
	host  string = "host"
	guest string = "guest"
)

func opgLabel(l labelKey) string {
	return opgLabelKeyPrefix + "/" + string(l)
}
