/**
 * Patient routes - approve or reject access requests
 */
const express = require('express');
const router = express.Router();
const { invoke } = require('../fabricClient');

/**
 * POST /patient/approve-request
 * Inputs: recordID, researcherID, expiryTimestamp
 */
router.post('/approve-request', async (req, res) => {
    try {
        const { recordID, researcherID, expiryTimestamp } = req.body;

        if (!recordID || !researcherID || !expiryTimestamp) {
            return res.status(400).json({ error: 'recordID, researcherID, and expiryTimestamp are required' });
        }

        // Patient can use any org identity - use hospital1 as patient proxy for demo
        const result = await invoke('hospital1', 'approveAccess', recordID, researcherID, String(expiryTimestamp));

        console.log(`[Patient] Consent granted for record ${recordID} and researcher ${researcherID}`);
        res.json({
            success: true,
            consent: result,
            message: 'Access approved'
        });
    } catch (err) {
        console.error('[Patient] Approve error:', err);
        res.status(500).json({ error: err.message });
    }
});

/**
 * POST /patient/reject-request
 * Input: recordID, researcherID
 */
router.post('/reject-request', async (req, res) => {
    try {
        const { recordID, researcherID } = req.body;

        if (!recordID || !researcherID) {
            return res.status(400).json({ error: 'recordID and researcherID are required' });
        }

        const result = await invoke('hospital1', 'rejectAccess', recordID, researcherID);

        console.log(`[Patient] Request rejected for record ${recordID} and researcher ${researcherID}`);
        res.json({
            success: true,
            request: result,
            message: 'Access rejected'
        });
    } catch (err) {
        console.error('[Patient] Reject error:', err);
        res.status(500).json({ error: err.message });
    }
});

module.exports = router;
