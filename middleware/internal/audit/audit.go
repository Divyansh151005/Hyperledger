package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/medical-middleware/internal/fabric"
	"github.com/hyperledger/medical-middleware/internal/storage"
	"github.com/sirupsen/logrus"
)

// AuditService handles audit logging
type AuditService struct {
	db *storage.Database
}

// NewAuditService creates a new audit service
func NewAuditService(db *storage.Database) *AuditService {
	return &AuditService{db: db}
}

// LogEvent logs an event to the audit database
func (s *AuditService) LogEvent(eventType, chaincodeName, transactionID string, blockNumber int64, invokerMSP, clientID string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.db.GetConnection().Exec(
		"INSERT INTO audit_logs (event_type, chaincode_name, transaction_id, block_number, invoker_msp, client_id, payload, timestamp) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		eventType, chaincodeName, transactionID, blockNumber, invokerMSP, clientID, payloadJSON, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs with optional filters
func (s *AuditService) GetAuditLogs(eventType, chaincodeName string, startTime, endTime *time.Time, limit, offset int) ([]AuditLog, error) {
	query := "SELECT id, event_type, chaincode_name, transaction_id, block_number, invoker_msp, client_id, payload, timestamp, created_at FROM audit_logs WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, eventType)
		argIndex++
	}

	if chaincodeName != "" {
		query += fmt.Sprintf(" AND chaincode_name = $%d", argIndex)
		args = append(args, chaincodeName)
		argIndex++
	}

	if startTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *startTime)
		argIndex++
	}

	if endTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *endTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC LIMIT $" + fmt.Sprintf("%d", argIndex) + " OFFSET $" + fmt.Sprintf("%d", argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.db.GetConnection().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var payloadJSON []byte
		if err := rows.Scan(&log.ID, &log.EventType, &log.ChaincodeName, &log.TransactionID, &log.BlockNumber, &log.InvokerMSP, &log.ClientID, &payloadJSON, &log.Timestamp, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &log.Payload); err != nil {
			log.Payload = make(map[string]interface{})
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            int64                  `json:"id"`
	EventType    string                 `json:"event_type"`
	ChaincodeName string                `json:"chaincode_name"`
	TransactionID string                `json:"transaction_id"`
	BlockNumber  int64                  `json:"block_number"`
	InvokerMSP   string                 `json:"invoker_msp"`
	ClientID     string                 `json:"client_id"`
	Payload      map[string]interface{} `json:"payload"`
	Timestamp    time.Time              `json:"timestamp"`
	CreatedAt    time.Time              `json:"created_at"`
}

// EventListener listens to Fabric events and logs them
type EventListener struct {
	gateway      *fabric.Gateway
	auditService *AuditService
	channelName  string
	stopChan     chan struct{}
}

// NewEventListener creates a new event listener
func NewEventListener(gateway *fabric.Gateway, auditService *AuditService, channelName string) (*EventListener, error) {
	return &EventListener{
		gateway:      gateway,
		auditService: auditService,
		channelName:  channelName,
		stopChan:     make(chan struct{}),
	}, nil
}

// Start starts listening to Fabric events
func (e *EventListener) Start(ctx context.Context) error {
	network := e.gateway.GetNetwork()

	// Listen to block events (network-wide)
	go e.listenToBlockEvents(ctx, network)

	// Listen to chaincode events from all chaincodes
	chaincodes := []string{"medical-records", "authorization-consent", "anonymous-sharing"}

	for _, chaincodeName := range chaincodes {
		go e.listenToChaincodeEvents(ctx, network, chaincodeName)
	}

	<-e.stopChan
	return nil
}

// Stop stops the event listener
func (e *EventListener) Stop() {
	close(e.stopChan)
}

// listenToBlockEvents listens to block events from the network
func (e *EventListener) listenToBlockEvents(ctx context.Context, network *client.Network) {
	events, err := network.BlockEvents(ctx)
	if err != nil {
		logrus.Errorf("Failed to register block events: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case block, ok := <-events:
			if !ok {
				logrus.Warn("Block events channel closed")
				return
			}
			e.processBlockEvent(block)
		}
	}
}

// processBlockEvent processes a Fabric block event
func (e *EventListener) processBlockEvent(block *common.Block) {
	// Extract block number
	blockNumber := int64(block.Header.Number)

	// Extract events from the block
	for range block.Data.Data {
		// Parse transaction envelope
		// In a real implementation, properly parse the transaction to extract events
		// For now, we'll log block-level information

		eventType := "BlockCommitted"
		transactionID := fmt.Sprintf("block-%d", blockNumber)

		payload := map[string]interface{}{
			"block_number": blockNumber,
		}

		// Extract invoker information if available
		invokerMSP := ""
		clientID := ""

		// Try to extract chaincode name from block if available
		chaincodeName := "unknown"

		if err := e.auditService.LogEvent(eventType, chaincodeName, transactionID, blockNumber, invokerMSP, clientID, payload); err != nil {
			logrus.Errorf("Failed to log block event: %v", err)
		}
	}
}

// listenToChaincodeEvents listens to chaincode-specific events (more detailed)
func (e *EventListener) listenToChaincodeEvents(ctx context.Context, network *client.Network, chaincodeName string) {
	events, err := network.ChaincodeEvents(ctx, chaincodeName)
	if err != nil {
		logrus.Errorf("Failed to register chaincode events for %s: %v", chaincodeName, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case event, ok := <-events:
			if !ok {
				logrus.Warnf("Chaincode events channel closed for %s", chaincodeName)
				return
			}
			e.processChaincodeEvent(chaincodeName, event)
		}
	}
}

// processChaincodeEvent processes a chaincode-specific event
func (e *EventListener) processChaincodeEvent(chaincodeName string, event *client.ChaincodeEvent) {
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logrus.Errorf("Failed to unmarshal event payload: %v", err)
		return
	}

	eventType := event.EventName
	transactionID := event.TransactionID

	invokerMSP := ""
	clientID := ""

	if msp, ok := payload["invokerMSP"].(string); ok {
		invokerMSP = msp
	}
	if cid, ok := payload["clientID"].(string); ok {
		clientID = cid
	}

	blockNumber := int64(0)
	if bn, ok := payload["blockNumber"].(float64); ok {
		blockNumber = int64(bn)
	}

	if err := e.auditService.LogEvent(eventType, chaincodeName, transactionID, blockNumber, invokerMSP, clientID, payload); err != nil {
		logrus.Errorf("Failed to log chaincode event: %v", err)
	}
}
