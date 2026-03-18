package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	authCCName     = "authorization-consent"
	medicalCCName  = "medical-records"
	sharedChannel  = "medical-main-channel"
	auditKeyPrefix = "AUDIT_"
)

type AnonymousSharingContract struct {
	contractapi.Contract
}

type AuditLog struct {
	LogID         string `json:"logID"`
	RecordID      string `json:"recordID"`
	RequestID     string `json:"requestID"`
	Action        string `json:"action"`
	Actor         string `json:"actor"`
	Details       string `json:"details"`
	AnonymizedRef string `json:"anonymizedRef,omitempty"`
	Timestamp     string `json:"timestamp"`
}

type AccessRequest struct {
	RequestID        string `json:"requestID"`
	RecordID         string `json:"recordID"`
	ResearcherID     string `json:"researcherID"`
	HospitalApproved bool   `json:"hospitalApproved"`
	PatientApproved  bool   `json:"patientApproved"`
	Expiry           int64  `json:"expiry"`
	Status           string `json:"status"`
}

type ChainRecord struct {
	RecordID string `json:"recordID"`
	Status   string `json:"status"`
}

func (c *AnonymousSharingContract) LogAnonymizedShare(
	ctx contractapi.TransactionContextInterface,
	recordID string,
	requestID string,
	actor string,
	anonymizedRef string,
	details string,
) (string, error) {
	if recordID == "" || requestID == "" || actor == "" {
		return "", fmt.Errorf("recordID, requestID and actor are required")
	}
	if _, err := queryRecord(ctx, recordID); err != nil {
		return "", err
	}
	req, err := queryAccessRequest(ctx, requestID)
	if err != nil {
		return "", err
	}
	if req.RecordID != recordID {
		return "", fmt.Errorf("request %s does not belong to record %s", requestID, recordID)
	}
	if req.Status != "ACTIVE" {
		return "", fmt.Errorf("request %s is not active", requestID)
	}
	if !req.HospitalApproved || !req.PatientApproved || req.Expiry < time.Now().UTC().UnixMilli() {
		return "", fmt.Errorf("invalid access state for anonymized sharing")
	}

	logID := fmt.Sprintf("%s%d", auditKeyPrefix, time.Now().UTC().UnixNano())
	log := AuditLog{
		LogID:         logID,
		RecordID:      recordID,
		RequestID:     requestID,
		Action:        "ANONYMIZED_SHARE",
		Actor:         actor,
		Details:       details,
		AnonymizedRef: anonymizedRef,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(log)
	if err != nil {
		return "", fmt.Errorf("failed to marshal audit log: %v", err)
	}
	if err := ctx.GetStub().PutState(logID, payload); err != nil {
		return "", fmt.Errorf("failed to write audit log: %v", err)
	}
	if err := ctx.GetStub().SetEvent("AnonymousShareLogged", payload); err != nil {
		return "", fmt.Errorf("failed to emit AnonymousShareLogged event: %v", err)
	}
	return logID, nil
}

func (c *AnonymousSharingContract) QueryAuditLogs(
	ctx contractapi.TransactionContextInterface,
) ([]*AuditLog, error) {
	it, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %v", err)
	}
	defer it.Close()

	logs := make([]*AuditLog, 0)
	for it.HasNext() {
		item, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("iterator failure: %v", err)
		}
		var log AuditLog
		if err := json.Unmarshal(item.Value, &log); err != nil {
			continue
		}
		if log.LogID == "" || log.RecordID == "" {
			continue
		}
		logs = append(logs, &log)
	}
	return logs, nil
}

func queryRecord(
	ctx contractapi.TransactionContextInterface,
	recordID string,
) (*ChainRecord, error) {
	response := ctx.GetStub().InvokeChaincode(
		medicalCCName,
		[][]byte{
			[]byte("QueryRecord"),
			[]byte(recordID),
		},
		sharedChannel,
	)
	if response.Status != shim.OK {
		return nil, fmt.Errorf("failed to fetch record: %s", response.Message)
	}
	var record ChainRecord
	if err := json.Unmarshal(response.Payload, &record); err != nil {
		return nil, fmt.Errorf("failed to parse medical-records response: %v", err)
	}
	if record.RecordID == "" {
		return nil, fmt.Errorf("record %s not found", recordID)
	}
	return &record, nil
}

func queryAccessRequest(
	ctx contractapi.TransactionContextInterface,
	requestID string,
) (*AccessRequest, error) {
	response := ctx.GetStub().InvokeChaincode(
		authCCName,
		[][]byte{
			[]byte("QueryRequest"),
			[]byte(requestID),
		},
		sharedChannel,
	)
	if response.Status != shim.OK {
		return nil, fmt.Errorf("failed to fetch access request: %s", response.Message)
	}
	var req AccessRequest
	if err := json.Unmarshal(response.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to parse authorization-consent response: %v", err)
	}
	if req.RequestID == "" {
		return nil, fmt.Errorf("request %s not found", requestID)
	}
	return &req, nil
}

func main() {
	cc, err := contractapi.NewChaincode(&AnonymousSharingContract{})
	if err != nil {
		fmt.Printf("error creating anonymous-sharing chaincode: %v", err)
		return
	}
	if err := cc.Start(); err != nil {
		fmt.Printf("error starting anonymous-sharing chaincode: %v", err)
	}
}
