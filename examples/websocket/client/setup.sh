CONNECTIONS=$1
REPLICAS=$2
IP=$3
go build --tags "static netgo" -o client client.go
for (( c=0; c<${REPLICAS}; c++ ))
do
    docker run -l 1m-go-websockets --network=dev-network -v ${LOCAL_WORKSPACE_FOLDER}/examples/websocket/client:/client -d alpine /client/client -conn=${CONNECTIONS} -ip=${IP} 
done