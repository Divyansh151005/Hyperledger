package fabric

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/hyperledger/medical-middleware/internal/config"
)

// Gateway wraps the Fabric Gateway client
type Gateway struct {
	conn    *grpc.ClientConn
	gateway *client.Gateway
	network *client.Network
}

// NewGateway creates a new Fabric Gateway connection
func NewGateway(cfg config.FabricConfig) (*Gateway, error) {
	// Load connection profile (simplified - in production, use proper connection profile parsing)
	// For now, we'll use environment variables or a simplified approach

	// Create gRPC connection to peer
	peerEndpoint := getEnv("FABRIC_PEER_ENDPOINT", "localhost:7051")
	peerTLSCertPath := getEnv("FABRIC_PEER_TLS_CERT", "")

	var opts []grpc.DialOption
	if peerTLSCertPath != "" {
		creds, err := credentials.NewClientTLSFromFile(peerTLSCertPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(peerEndpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	// Load identity
	id, err := loadIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	// Create signer
	sign, err := loadSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to load signer: %w", err)
	}

	// Create gateway
	gateway, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway: %w", err)
	}

	// Get network
	network := gateway.GetNetwork(cfg.ChannelName)

	return &Gateway{
		conn:    conn,
		gateway: gateway,
		network: network,
	}, nil
}

// Close closes the gateway connection
func (g *Gateway) Close() error {
	if g.gateway != nil {
		g.gateway.Close()
	}
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

// GetNetwork returns the network client
func (g *Gateway) GetNetwork() *client.Network {
	return g.network
}

// GetContract returns a contract client for the given chaincode
func (g *Gateway) GetContract(chaincodeName string) *client.Contract {
	return g.network.GetContract(chaincodeName)
}

// loadIdentity loads the client identity from certificate
func loadIdentity() (*identity.X509Identity, error) {
	certPath := getEnv("FABRIC_CERT_PATH", "./crypto-config/peerOrganizations/hospital1.medical-network.com/users/User1@hospital1.medical-network.com/msp/signcerts/User1@hospital1.medical-network.com-cert.pem")
	keyPath := getEnv("FABRIC_KEY_PATH", "./crypto-config/peerOrganizations/hospital1.medical-network.com/users/User1@hospital1.medical-network.com/msp/keystore")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	// Parse the X.509 certificate
	cert, err := ParseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Find private key (for signer, not needed here but kept for reference)
	keyFiles, err := filepath.Glob(filepath.Join(keyPath, "*_sk"))
	if err != nil || len(keyFiles) == 0 {
		return nil, fmt.Errorf("failed to find private key")
	}

	key, err := os.ReadFile(keyFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Create identity with parsed certificate
	id, err := identity.NewX509Identity("HospitalMSP1", cert)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	// Note: Private key is used for signer, not identity
	_ = key

	return id, nil
}

// loadSigner loads the signer for transactions
func loadSigner() (identity.Sign, error) {
	keyPath := getEnv("FABRIC_KEY_PATH", "./crypto-config/peerOrganizations/hospital1.medical-network.com/users/User1@hospital1.medical-network.com/msp/keystore")

	keyFiles, err := filepath.Glob(filepath.Join(keyPath, "*_sk"))
	if err != nil || len(keyFiles) == 0 {
		return nil, fmt.Errorf("failed to find private key")
	}

	key, err := os.ReadFile(keyFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	privateKey, err := identity.PrivateKeyFromPEM(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// NewPrivateKeySign returns (Sign, error)
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return sign, nil
}

// getEnv gets an environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ExtractError extracts error message from gRPC status
func ExtractError(err error) error {
	if err == nil {
		return nil
	}
	if s, ok := status.FromError(err); ok {
		return fmt.Errorf("fabric error: %s", s.Message())
	}
	return err
}

// InvokeContract invokes a chaincode function
func InvokeContract(ctx context.Context, contract *client.Contract, function string, args ...string) ([]byte, error) {
	proposal, err := contract.NewProposal(function, client.WithArguments(args...))
	if err != nil {
		return nil, fmt.Errorf("failed to create proposal: %w", err)
	}

	transaction, err := proposal.Endorse()
	if err != nil {
		return nil, fmt.Errorf("failed to endorse transaction: %w", err)
	}

	// Get result from transaction before submitting
	result := transaction.Result()

	commit, err := transaction.Submit()
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Wait for commit status and check for success
	status, err := commit.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit status: %w", err)
	}

	if !status.Successful {
		return nil, fmt.Errorf("transaction commit failed with status: %v", status)
	}

	return result, nil
}

// QueryContract queries a chaincode function
func QueryContract(ctx context.Context, contract *client.Contract, function string, args ...string) ([]byte, error) {
	result, err := contract.Evaluate(function, client.WithArguments(args...))
	if err != nil {
		return nil, ExtractError(err)
	}
	return result, nil
}

// ParseCertificate parses an X.509 certificate
func ParseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
