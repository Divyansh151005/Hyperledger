/**
 * Regulator routes - audit and revoke consent
 */
const express = require('express');
const router = express.Router();
const { invoke, query } = require('../fabricClient');

/**
 * GET /regulator/audit-records
 */
router.get('/audit-records', async (req, res) => {
    try {
        const audit = await query('regulator', 'auditRecords');
        const records = audit && Array.isArray(audit.records) ? audit.records : [];
        const requests = audit && Array.isArray(audit.requests) ? audit.requests : [];
        const consents = audit && Array.isArray(audit.consents) ? audit.consents : [];

        console.log(`[Regulator] Audited records=${records.length}, requests=${requests.length}, consents=${consents.length}`);
        res.json({
            success: true,
            records,
            requests,
            consents
        });
    } catch (err) {
        console.error('[Regulator] Audit records error:', err);
        res.status(500).json({ error: err.message });
    }
});

/**
 * GET /regulator/audit-consents
 */
router.get('/audit-consents', async (req, res) => {
    try {
        const consents = await query('regulator', 'getAllConsents');

        console.log(`[Regulator] Audited ${Array.isArray(consents) ? consents.length : 0} consents`);
        res.json({
            success: true,
            consents: consents || []
        });
    } catch (err) {
        console.error('[Regulator] Audit consents error:', err);
        res.status(500).json({ error: err.message });
    }
});

/**
 * POST /regulator/revoke-consent
 * Input: recordID, researcherID
 */
router.post('/revoke-consent', async (req, res) => {
    try {
        const { recordID, researcherID } = req.body;

        if (!recordID || !researcherID) {
            return res.status(400).json({ error: 'recordID and researcherID are required' });
        }

        const revoked = await invoke('regulator', 'revokeConsent', recordID, researcherID);

        console.log(`[Regulator] Consent revoked for record ${recordID} and researcher ${researcherID}`);
        res.json({
            success: true,
            consent: revoked,
            message: 'Consent revoked'
        });
    } catch (err) {
        console.error('[Regulator] Revoke error:', err);
        res.status(500).json({ error: err.message });
    }
});

module.exports = router;
