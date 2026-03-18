package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	medicalRecordsCC = "medical-records"
	channelName      = "medical-main-channel"
)

const (
	RequestStatusPending = "PENDING"
	RequestStatusActive  = "ACTIVE"
	RequestStatusRevoked = "REVOKED"
)

type AuthorizationConsentContract struct {
	contractapi.Contract
}

type AccessRequest struct {
	RequestID        string `json:"requestID"`
	RecordID         string `json:"recordID"`
	ResearcherID     string `json:"researcherID"`
	PatientID        string `json:"patientID"`
	HospitalID       string `json:"hospitalID"`
	HospitalApproved bool   `json:"hospitalApproved"`
	PatientApproved  bool   `json:"patientApproved"`
	Expiry           int64  `json:"expiry"`
	Status           string `json:"status"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type AccessEvent struct {
	Action       string `json:"action"`
	RequestID    string `json:"requestID"`
	RecordID     string `json:"recordID"`
	ResearcherID string `json:"researcherID,omitempty"`
	Status       string `json:"status"`
	Expiry       int64  `json:"expiry,omitempty"`
	Timestamp    string `json:"timestamp"`
}

type ChainRecord struct {
	RecordID   string `json:"recordID"`
	PatientID  string `json:"patientID"`
	HospitalID string `json:"hospitalID"`
	Status     string `json:"status"`
}

func (c *AuthorizationConsentContract) RequestAccess(
	ctx contractapi.TransactionContextInterface,
	recordID string,
	researcherID string,
) (string, error) {
	if recordID == "" || researcherID == "" {
		return "", fmt.Errorf("recordID and researcherID are required")
	}

	record, err := queryRecordFromMedicalCC(ctx, recordID)
	if err != nil {
		return "", err
	}
	if record.Status != "APPROVED" {
		return "", fmt.Errorf("record %s is not approved for access flow", recordID)
	}

	requestID := fmt.Sprintf("REQ-%d", time.Now().UTC().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	req := AccessRequest{
		RequestID:        requestID,
		RecordID:         recordID,
		ResearcherID:     researcherID,
		PatientID:        record.PatientID,
		HospitalID:       record.HospitalID,
		HospitalApproved: false,
		PatientApproved:  false,
		Expiry:           0,
		Status:           RequestStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := putRequest(ctx, &req); err != nil {
		return "", err
	}
	if err := emitEvent(ctx, "AccessRequested", AccessEvent{
		Action:       "AccessRequested",
		RequestID:    requestID,
		RecordID:     recordID,
		ResearcherID: researcherID,
		Status:       req.Status,
		Timestamp:    now,
	}); err != nil {
		return "", err
	}
	return requestID, nil
}

func (c *AuthorizationConsentContract) ApproveByHospital(
	ctx contractapi.TransactionContextInterface,
	requestID string,
) error {
	req, err := getRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if req.Status == RequestStatusRevoked {
		return fmt.Errorf("request %s is revoked", requestID)
	}

	req.HospitalApproved = true
	req.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	req.Status = deriveStatus(req)

	if err := putRequest(ctx, req); err != nil {
		return err
	}
	if err := emitEvent(ctx, "AccessApproved", AccessEvent{
		Action:       "ApproveByHospital",
		RequestID:    req.RequestID,
		RecordID:     req.RecordID,
		ResearcherID: req.ResearcherID,
		Status:       req.Status,
		Expiry:       req.Expiry,
		Timestamp:    req.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (c *AuthorizationConsentContract) ApproveByPatient(
	ctx contractapi.TransactionContextInterface,
	requestID string,
	expiry string,
) error {
	req, err := getRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if req.Status == RequestStatusRevoked {
		return fmt.Errorf("request %s is revoked", requestID)
	}

	expiryTS, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil || expiryTS <= time.Now().UTC().UnixMilli() {
		return fmt.Errorf("expiry must be a future unix millisecond timestamp")
	}

	req.PatientApproved = true
	req.Expiry = expiryTS
	req.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	req.Status = deriveStatus(req)

	if err := putRequest(ctx, req); err != nil {
		return err
	}
	if err := emitEvent(ctx, "AccessApproved", AccessEvent{
		Action:       "ApproveByPatient",
		RequestID:    req.RequestID,
		RecordID:     req.RecordID,
		ResearcherID: req.ResearcherID,
		Status:       req.Status,
		Expiry:       req.Expiry,
		Timestamp:    req.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (c *AuthorizationConsentContract) RevokeAccess(
	ctx contractapi.TransactionContextInterface,
	requestID string,
) error {
	req, err := getRequest(ctx, requestID)
	if err != nil {
		return err
	}
	req.Status = RequestStatusRevoked
	req.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := putRequest(ctx, req); err != nil {
		return err
	}

	if err := emitEvent(ctx, "AccessRevoked", AccessEvent{
		Action:       "RevokeAccess",
		RequestID:    req.RequestID,
		RecordID:     req.RecordID,
		ResearcherID: req.ResearcherID,
		Status:       req.Status,
		Timestamp:    req.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (c *AuthorizationConsentContract) QueryRequests(
	ctx contractapi.TransactionContextInterface,
) ([]*AccessRequest, error) {
	it, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to query requests: %v", err)
	}
	defer it.Close()

	requests := make([]*AccessRequest, 0)
	for it.HasNext() {
		item, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("iterator failure: %v", err)
		}
		var req AccessRequest
		if err := json.Unmarshal(item.Value, &req); err != nil {
			continue
		}
		if req.RequestID == "" || req.RecordID == "" {
			continue
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func (c *AuthorizationConsentContract) QueryRequest(
	ctx contractapi.TransactionContextInterface,
	requestID string,
) (*AccessRequest, error) {
	return getRequest(ctx, requestID)
}

func queryRecordFromMedicalCC(
	ctx contractapi.TransactionContextInterface,
	recordID string,
) (*ChainRecord, error) {
	response := ctx.GetStub().InvokeChaincode(
		medicalRecordsCC,
		[][]byte{
			[]byte("QueryRecord"),
			[]byte(recordID),
		},
		channelName,
	)
	if response.Status != shim.OK {
		return nil, fmt.Errorf("failed to fetch record from medical-records: %s", response.Message)
	}

	var record ChainRecord
	if err := json.Unmarshal(response.Payload, &record); err != nil {
		return nil, fmt.Errorf("failed to parse medical-records payload: %v", err)
	}
	if record.RecordID == "" {
		return nil, fmt.Errorf("record %s not found in medical-records", recordID)
	}
	return &record, nil
}

func getRequest(
	ctx contractapi.TransactionContextInterface,
	requestID string,
) (*AccessRequest, error) {
	if requestID == "" {
		return nil, fmt.Errorf("requestID is required")
	}
	raw, err := ctx.GetStub().GetState(requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to read request: %v", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("request %s does not exist", requestID)
	}
	var req AccessRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %v", err)
	}
	return &req, nil
}

func putRequest(
	ctx contractapi.TransactionContextInterface,
	req *AccessRequest,
) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}
	if err := ctx.GetStub().PutState(req.RequestID, payload); err != nil {
		return fmt.Errorf("failed to write request: %v", err)
	}
	return nil
}

func deriveStatus(req *AccessRequest) string {
	if req.Status == RequestStatusRevoked {
		return RequestStatusRevoked
	}
	if req.HospitalApproved && req.PatientApproved && req.Expiry > time.Now().UTC().UnixMilli() {
		return RequestStatusActive
	}
	return RequestStatusPending
}

func emitEvent(
	ctx contractapi.TransactionContextInterface,
	eventName string,
	payload AccessEvent,
) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}
	if err := ctx.GetStub().SetEvent(eventName, raw); err != nil {
		return fmt.Errorf("failed to emit %s event: %v", eventName, err)
	}
	return nil
}

func main() {
	cc, err := contractapi.NewChaincode(&AuthorizationConsentContract{})
	if err != nil {
		fmt.Printf("error creating authorization-consent chaincode: %v", err)
		return
	}
	if err := cc.Start(); err != nil {
		fmt.Printf("error starting authorization-consent chaincode: %v", err)
	}
}
