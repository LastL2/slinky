
# this script is for generating protobuf files for the new google.golang.org/protobuf API

set -eo pipefail

protoc_install_gopulsar() {
  go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
}

protoc_install_gopulsar

echo "Cleaning API directory"
(cd api; find ./ -type f \( -iname \*.pulsar.go -o -iname \*.pb.go -o -iname \*.cosmos_orm.go -o -iname \*.pb.gw.go \) -delete; find . -empty -type d -delete; cd ..)

echo "Generating API module"
(cd proto; buf generate --template buf.gen.pulsar.yaml)

echo "fixing alerts.pulsar.go"
sed -i.bak 's|cosmossdk.io/api/slinky/oracle/v1|github.com/skip-mev/slinky/api/slinky/oracle/v1|g' ./api/slinky/alerts/v1/alerts.pulsar.go && rm ./api/slinky/alerts/v1/alerts.pulsar.go.bak

echo "fixing sla.pulsar.go"
sed -i.bak 's|cosmossdk.io/api/slinky/oracle/v1|github.com/skip-mev/slinky/api/slinky/oracle/v1|g' ./api/slinky/sla/v1/genesis.pulsar.go && rm ./api/slinky/sla/v1/genesis.pulsar.go.bak