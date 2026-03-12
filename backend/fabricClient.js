/**
 * Fabric Gateway client - connects to peer and invokes chaincode
 */
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { connect, hash, signers } = require('@hyperledger/fabric-gateway');
const grpc = require('@grpc/grpc-js');

const CHANNEL_NAME = process.env.FABRIC_CHANNEL_NAME || 'medical-main-channel';
const CHAINCODE_NAME = process.env.FABRIC_CHAINCODE_NAME || 'medicalcc';
const CRYPTO_PATH = process.env.CRYPTO_PATH || path.join(__dirname, '../network/crypto-config');
const PEER_HOST = process.env.PEER_HOST || 'localhost';
const PEER_PORT = process.env.PEER_PORT || '11051'; // regulator peer
const TLS_CACERT = process.env.TLS_CACERT || path.join(
    CRYPTO_PATH,
    'peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt'
);

/**
 * Load identity for an organization
 */
function loadIdentity(org, user = 'Admin') {
    const orgDomain = {
        hospital1: 'hospital1.medical-network.com',
        hospital2: 'hospital2.medical-network.com',
        research: 'research.medical-network.com',
        regulator: 'regulator.medical-network.com'
    }[org] || org;

    const certPath = path.join(
        CRYPTO_PATH,
        'peerOrganizations', orgDomain,
        'users', `${user}@${orgDomain}`, 'msp', 'signcerts',
        `${user}@${orgDomain}-cert.pem`
    );
    return fs.readFileSync(certPath);
}

/**
 * Load private key for an organization
 */
function loadPrivateKey(org, user = 'Admin') {
    const orgDomain = {
        hospital1: 'hospital1.medical-network.com',
        hospital2: 'hospital2.medical-network.com',
        research: 'research.medical-network.com',
        regulator: 'regulator.medical-network.com'
    }[org] || org;

    const keyDir = path.join(
        CRYPTO_PATH,
        'peerOrganizations', orgDomain,
        'users', `${user}@${orgDomain}`, 'msp', 'keystore'
    );
    const files = fs.readdirSync(keyDir);
    const keyFile = files.find(f => f.endsWith('_sk'));
    if (!keyFile) throw new Error(`No private key found in ${keyDir}`);
    return fs.readFileSync(path.join(keyDir, keyFile));
}

/**
 * Create gRPC connection to peer
 */
function createGrpcConnection() {
    const tlsRootCert = fs.existsSync(TLS_CACERT)
        ? fs.readFileSync(TLS_CACERT)
        : Buffer.from('');
    const tlsCredentials = tlsRootCert.length > 0
        ? grpc.credentials.createSsl(tlsRootCert)
        : grpc.credentials.createInsecure();
    return new grpc.Client(
        `${PEER_HOST}:${PEER_PORT}`,
        tlsCredentials,
        { 'grpc.max_receive_message_length': -1 }
    );
}

const MSP_IDS = {
    hospital1: 'HospitalMSP1',
    hospital2: 'HospitalMSP2',
    research: 'ResearchOrgMSP',
    regulator: 'RegulatorMSP'
};

/**
 * Create Gateway with identity
 */
async function createGateway(org = 'hospital1') {
    const client = createGrpcConnection();
    const credentials = loadIdentity(org);
    const privateKeyPem = loadPrivateKey(org);
    const privateKey = crypto.createPrivateKey(privateKeyPem);
    const signer = signers.newPrivateKeySigner(privateKey);

    const gateway = connect({
        client,
        identity: {
            mspId: MSP_IDS[org] || 'HospitalMSP1',
            credentials
        },
        signer,
        hash: hash.sha256
    });

    return gateway;
}

/**
 * Invoke chaincode (submit transaction)
 */
async function invoke(org, fnName, ...args) {
    const gateway = await createGateway(org);
    try {
        const network = gateway.getNetwork(CHANNEL_NAME);
        const contract = network.getContract(CHAINCODE_NAME);
        const proposal = contract.newProposal(fnName, {
            arguments: args,
            endorsingOrganizations: ['HospitalMSP1']
        });
        const endorsed = await proposal.endorse();
        const result = await endorsed.submit();
        return result ? JSON.parse(result.toString()) : {};
    } finally {
        gateway.close();
    }
}

/**
 * Query chaincode (evaluate - no commit)
 */
async function query(org, fnName, ...args) {
    const gateway = await createGateway(org);
    try {
        const network = gateway.getNetwork(CHANNEL_NAME);
        const contract = network.getContract(CHAINCODE_NAME);
        const result = await contract.evaluateTransaction(fnName, ...args);
        return result ? JSON.parse(result.toString()) : null;
    } finally {
        gateway.close();
    }
}

/**
 * Listen for chaincode events
 */
async function listenForEvents(callback) {
    const gateway = await createGateway('regulator');
    const network = gateway.getNetwork(CHANNEL_NAME);
    const events = await network.newChaincodeEventsRequest(CHAINCODE_NAME).getEvents();
    (async () => {
        try {
            for await (const event of events) {
                try {
                    const payload = event.payload && event.payload.length > 0
                        ? JSON.parse(event.payload.toString())
                        : {};
                    callback(event.eventName, payload);
                } catch (parseErr) {
                    callback(event.eventName, { raw: event.payload?.toString() });
                }
            }
        } catch (err) {
            console.error('[Event Listener] Error:', err.message);
        }
    })();
    return gateway;
}

module.exports = {
    createGateway,
    invoke,
    query,
    listenForEvents,
    CHANNEL_NAME,
    CHAINCODE_NAME
};
