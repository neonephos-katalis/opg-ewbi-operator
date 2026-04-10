#!/bin/bash

# Generate the code for the API and callbacks
api=./swagger.yaml

cd /api/federation
yq eval-all --inplace 'del(.servers, .components.securitySchemes, .security) |
        ... comments="" |
        . head_comment="DO NOT EDIT - Source: https://github.com/edge-collab/federation-ewbi" ' $api
oapi-codegen --config=models.cfg.yaml $api
oapi-codegen --config=server.cfg.yaml $api
oapi-codegen --config=client.cfg.yaml $api
## Manual fix for this oapi-codegen bugs:
## https://github.com/deepmap/oapi-codegen/issues/899
## https://github.com/deepmap/oapi-codegen/issues/399
sed -i 's/N200AppDeploymentZonesZoneInfoResourceConsumption/InstallAppJSONBodyZoneInfoResourceConsumption/g' ./client/client.gen.go
sed -i 's/N200AppMetaDataCategory/OnboardApplicationJSONBodyAppMetaDataCategory/g' ./client/client.gen.go
sed -i 's/N200AppQoSProfileMultiUserClients/OnboardApplicationJSONBodyAppQoSProfileMultiUserClients/g' ./client/client.gen.go
sed -i 's/N200ArtefactDescriptorType/UploadArtefactMultipartBodyArtefactDescriptorType/g' ./client/client.gen.go
sed -i 's/N200ArtefactFileFormat/UploadArtefactMultipartBodyArtefactFileFormat/g' ./client/client.gen.go
sed -i 's/N200ArtefactVirtType/UploadArtefactMultipartBodyArtefactVirtType/g' ./client/client.gen.go
sed -i 's/N200AppQoSProfileLatencyConstraints/OnboardApplicationJSONBodyAppQoSProfileLatencyConstraints/g' ./client/client.gen.go
sed -i 's/N200RepoType/UploadArtefactMultipartBodyRepoType/g' ./client/client.gen.go
sed -i 's/GetArtefact200JSONResponseArtefactDescriptorType/UploadArtefactMultipartBodyArtefactDescriptorType/g' ./server/server.gen.go
sed -i 's/GetArtefact200JSONResponseArtefactFileFormat/UploadArtefactMultipartBodyArtefactFileFormat/g' ./server/server.gen.go
sed -i 's/GetArtefact200JSONResponseArtefactVirtType/UploadArtefactMultipartBodyArtefactVirtType/g' ./server/server.gen.go
sed -i 's/GetArtefact200JSONResponseRepoType/UploadArtefactMultipartBodyRepoType/g' ./server/server.gen.go
sed -i 's/ViewFile200JSONResponseRepoType/UploadFileMultipartBodyRepoType/g' ./server/server.gen.go
sed -i 's/N200ArtefactDescriptorType/UploadArtefactMultipartBodyArtefactDescriptorType/g' ./client/client.gen.go
sed -i 's/N200ArtefactFileFormat/UploadArtefactMultipartBodyArtefactFileFormat/g' ./client/client.gen.go
sed -i 's/N200ArtefactVirtType/UploadArtefactMultipartBodyArtefactVirtType/g' ./client/client.gen.go
sed -i 's/N200RepoType/UploadArtefactMultipartBodyRepoType/g' ./client/client.gen.go
