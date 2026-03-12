FROM hyperledger/fabric-baseos:2.5

WORKDIR /opt/gopath/src/github.com/chaincode

COPY . .

RUN go mod tidy
RUN go build -o chaincode .

CMD ["./chaincode"]
